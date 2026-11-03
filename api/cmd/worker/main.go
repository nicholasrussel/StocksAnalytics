package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tripleboys27/stock-dashboard/api/internal/fmp"
	"github.com/tripleboys27/stock-dashboard/api/internal/llm"
	"github.com/tripleboys27/stock-dashboard/api/internal/news"
	"github.com/tripleboys27/stock-dashboard/api/internal/scorecard"
	"github.com/tripleboys27/stock-dashboard/api/internal/store"
	"github.com/tripleboys27/stock-dashboard/api/internal/telegram"
)

func main() {
	_ = godotenv.Load()

	jobs := flag.String("jobs", "", "comma-separated list of jobs to run")
	flag.Parse()

	if *jobs == "" {
		slog.Error("no jobs specified")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	bot := telegram.NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"))
	finnhub := news.NewFinnhubClient(os.Getenv("FINNHUB_API_KEY"))
	gemini := llm.NewGeminiClient(os.Getenv("GEMINI_API_KEY"), os.Getenv("LLM_MODEL"))

	var fmpClient *fmp.Client

	if key := os.Getenv("FMP_API_KEY"); key != "" {
		fmpClient = fmp.NewClient(key)
	}

	for _, job := range strings.Split(*jobs, ",") {
		job = strings.TrimSpace(job)
		if job == "" {
			continue
		}
		slog.Info("running job", "job", job)

		switch job {
		case "bot-poll":
			runBotPoll(ctx, db, bot)
		case "news":
			runNews(ctx, db, finnhub)
		case "analyze":
			runAnalyze(ctx, db, gemini)
		case "notify":
			runNotify(ctx, db, bot)
		case "fundamentals":
			runFundamentals(ctx, db, fmpClient, gemini, finnhub)
		case "market-notify":
			runMarketNotify(ctx, db, bot, finnhub)
		case "sector-notify":
			runSectorNotify(ctx, db, bot, finnhub)
		default:
			slog.Warn("unknown job", "job", job)
		}
	}
}

// ── bot-poll ──────────────────────────────────────────────────────────────────

func runBotPoll(ctx context.Context, db *store.DB, bot *telegram.Bot) {
	slog.Info("telegram bot polling started — tekan Ctrl+C untuk berhenti")
	var offset int64
	for {
		select {
		case <-ctx.Done():
			slog.Info("bot poll stopped")
			return
		default:
		}

		updates, err := bot.GetUpdates(ctx, offset)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			slog.Error("getUpdates error", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		for _, update := range updates {
			if update.ID >= offset {
				offset = update.ID + 1
			}
			if update.Message == nil {
				continue
			}
			slog.Info("message received", "chat_id", update.Message.Chat.ID, "text", update.Message.Text)
			handleMessage(ctx, db, bot, update.Message)
		}
	}
}

func handleMessage(ctx context.Context, db *store.DB, bot *telegram.Bot, msg *telegram.Message) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID

	if !strings.HasPrefix(text, "/start") {
		return
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		_ = bot.SendMessage(ctx, chatID, "Gunakan link dari aplikasi web untuk menghubungkan akun kamu.")
		return
	}

	token := parts[1]
	user, err := db.UseLinkToken(ctx, token, chatID)
	if err != nil {
		var reply string
		switch err {
		case store.ErrTokenNotFound:
			reply = "❌ Token tidak ditemukan. Silakan buat link baru dari aplikasi."
		case store.ErrTokenUsed:
			reply = "❌ Token sudah pernah digunakan."
		case store.ErrTokenExpired:
			reply = "❌ Token sudah kedaluwarsa. Silakan buat link baru dari aplikasi."
		default:
			reply = fmt.Sprintf("❌ Terjadi kesalahan: %s", err.Error())
		}
		_ = bot.SendMessage(ctx, chatID, reply)
		slog.Warn("link token error", "token", token, "err", err)
		return
	}

	name := user.Email
	if user.Name != nil && *user.Name != "" {
		name = *user.Name
	}
	reply := fmt.Sprintf("✅ Berhasil terhubung!\n\nHai <b>%s</b>, akun Telegram kamu sudah terhubung ke Stock Research Dashboard.\nKamu akan mendapat notifikasi berita penting sesuai watchlist.", name)
	_ = bot.SendMessage(ctx, chatID, reply)
	slog.Info("telegram linked", "user", user.Email, "chat_id", chatID)
}

// ── news ──────────────────────────────────────────────────────────────────────

func runNews(ctx context.Context, db *store.DB, finnhub *news.FinnhubClient) {
	tickers, err := db.GetAllWatchlistTickers(ctx)
	if err != nil {
		slog.Error("news: failed to get tickers", "err", err)
		return
	}
	if len(tickers) == 0 {
		slog.Info("news: no tickers in any watchlist")
		return
	}

	to := time.Now()
	from := to.Add(-24 * time.Hour)
	var inserted, skipped int

	for _, ticker := range tickers {
		select {
		case <-ctx.Done():
			return
		default:
		}

		items, err := finnhub.FetchCompanyNews(ctx, ticker, from, to)
		if err != nil {
			slog.Warn("news: fetch failed", "ticker", ticker, "err", err)
			continue
		}

		for _, item := range items {
			if item.URL == "" || item.Headline == "" {
				continue
			}
			isNew, err := db.UpsertNewsItem(ctx, ticker,
				item.Headline, item.URL, item.Summary,
				"finnhub", item.Source, item.PublishedAt,
			)
			if err != nil {
				slog.Warn("news: upsert failed", "ticker", ticker, "err", err)
				continue
			}
			if isNew {
				inserted++
			} else {
				skipped++
			}
		}
		slog.Info("news: fetched", "ticker", ticker, "count", len(items))
	}

	slog.Info("news: done", "inserted", inserted, "skipped_duplicates", skipped)
}

// ── analyze ───────────────────────────────────────────────────────────────────

func runAnalyze(ctx context.Context, db *store.DB, gemini *llm.GeminiClient) {
	items, err := db.GetUnanalyzedNews(ctx)
	if err != nil {
		slog.Error("analyze: failed to get unanalyzed news", "err", err)
		return
	}
	if len(items) == 0 {
		slog.Info("analyze: nothing to analyze")
		return
	}
	slog.Info("analyze: items to process", "count", len(items))

	var done, failed int
	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		default:
		}

		analysis, err := gemini.AnalyzeSingleNews(ctx, item.Ticker, item.Title, item.Snippet)
		if err != nil {
			slog.Warn("analyze: gemini failed", "id", item.ID, "ticker", item.Ticker, "err", err)
			failed++
			continue
		}

		if err := db.MarkNewsAnalyzed(ctx, item.ID, analysis.Importance, analysis.Sentiment, analysis.Summary); err != nil {
			slog.Warn("analyze: db update failed", "id", item.ID, "err", err)
			failed++
			continue
		}
		done++
	}

	slog.Info("analyze: done", "analyzed", done, "failed", failed)
}

// ── notify ────────────────────────────────────────────────────────────────────

func runNotify(ctx context.Context, db *store.DB, bot *telegram.Bot) {
	candidates, err := db.GetNotifyCandidates(ctx)
	if err != nil {
		slog.Error("notify: failed to get candidates", "err", err)
		return
	}
	if len(candidates) == 0 {
		slog.Info("notify: nothing to send")
		return
	}

	// Group by user
	type userGroup struct {
		chatID     int64
		userID     string
		candidates []store.NotifyCandidate
	}
	groups := map[string]*userGroup{}
	for _, c := range candidates {
		g, ok := groups[c.UserID]
		if !ok {
			g = &userGroup{chatID: c.ChatID, userID: c.UserID}
			groups[c.UserID] = g
		}
		g.candidates = append(g.candidates, c)
	}

	sentTotal := 0
	for _, g := range groups {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Cap at 5 items per message to keep it readable
		items := g.candidates
		if len(items) > 5 {
			items = items[:5]
		}

		msg := buildNotifyMessage(items)
		if err := bot.SendMessage(ctx, g.chatID, msg); err != nil {
			slog.Warn("notify: send failed", "user", g.userID, "err", err)
			continue
		}

		for _, c := range items {
			if err := db.RecordNotificationSent(ctx, g.userID, c.NewsID); err != nil {
				slog.Warn("notify: record failed", "user", g.userID, "news_id", c.NewsID, "err", err)
			}
		}
		sentTotal += len(items)
		slog.Info("notify: sent", "user", g.userID, "items", len(items))
	}

	slog.Info("notify: done", "total_sent", sentTotal)
}

func buildNotifyMessage(items []store.NotifyCandidate) string {
	var sb strings.Builder
	sb.WriteString("📊 <b>Alert Watchlist Kamu</b>\n\n")

	for _, item := range items {
		sentimentEmoji := "🟡"
		switch item.Sentiment {
		case "positif":
			sentimentEmoji = "🟢"
		case "negatif":
			sentimentEmoji = "🔴"
		}

		sb.WriteString(fmt.Sprintf("<b>%s</b> %s %s | Penting: %d/5\n", item.Ticker, sentimentEmoji, item.Sentiment, item.Importance))
		sb.WriteString(fmt.Sprintf("<i>%s</i>\n", escapeHTML(item.Title)))
		if item.Summary != "" {
			sb.WriteString(item.Summary + "\n")
		}
		sb.WriteString(fmt.Sprintf("🔗 %s\n\n", item.URL))
	}

	return strings.TrimRight(sb.String(), "\n")
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// ── fundamentals ──────────────────────────────────────────────────────────────

func runFundamentals(ctx context.Context, db *store.DB, fmpClient *fmp.Client, gemini *llm.GeminiClient, finnhub *news.FinnhubClient) {
	if fmpClient == nil {
		slog.Warn("fundamentals: FMP_API_KEY not set, skipping")
		return
	}

	tickers, err := db.GetAllWatchlistTickers(ctx)
	if err != nil {
		slog.Error("fundamentals: get tickers", "err", err)
		return
	}
	if len(tickers) == 0 {
		slog.Info("fundamentals: no tickers in any watchlist")
		return
	}
	slog.Info("fundamentals: processing tickers", "count", len(tickers))

	staleDays := 7 * 24 * time.Hour
	var refreshed, skipped, failed int

	for _, ticker := range tickers {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Skip if scorecard is still fresh
		cached, err := db.GetLatestScorecard(ctx, ticker)
		if err == nil && time.Since(cached.ComputedAt) < staleDays {
			slog.Info("fundamentals: skipping fresh scorecard", "ticker", ticker,
				"age_hours", int(time.Since(cached.ComputedAt).Hours()))
			skipped++
			continue
		}

		// Fetch FMP data concurrently
		var (
			incomes   []fmp.IncomeStatement
			balances  []fmp.BalanceSheet
			cashflows []fmp.CashFlowStatement
			quote     *fmp.Quote
			fetchErr  error
			mu        sync.Mutex
			wg        sync.WaitGroup
		)

		fetch := func(fn func() error) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if e := fn(); e != nil {
					mu.Lock()
					fetchErr = e
					mu.Unlock()
				}
			}()
		}

		fetch(func() error {
			v, e := fmpClient.GetIncomeStatements(ctx, ticker, 2)
			if e == nil {
				mu.Lock()
				incomes = v
				mu.Unlock()
			}
			return e
		})
		fetch(func() error {
			v, e := fmpClient.GetBalanceSheets(ctx, ticker, 2)
			if e == nil {
				mu.Lock()
				balances = v
				mu.Unlock()
			}
			return e
		})
		fetch(func() error {
			v, e := fmpClient.GetCashFlowStatements(ctx, ticker, 2)
			if e == nil {
				mu.Lock()
				cashflows = v
				mu.Unlock()
			}
			return e
		})
		fetch(func() error {
			v, e := fmpClient.GetQuote(ctx, ticker)
			if e == nil {
				mu.Lock()
				quote = v
				mu.Unlock()
			}
			return e
		})
		wg.Wait()

		if fetchErr != nil || len(incomes) == 0 || len(balances) == 0 || len(cashflows) == 0 {
			slog.Warn("fundamentals: fetch failed", "ticker", ticker, "err", fetchErr)
			failed++
			continue
		}

		var prevIncome *fmp.IncomeStatement
		var prevBalance *fmp.BalanceSheet
		if len(incomes) > 1 {
			prevIncome = &incomes[1]
		}
		if len(balances) > 1 {
			prevBalance = &balances[1]
		}

		sc := scorecard.Compute(ticker, incomes[0], balances[0], cashflows[0], quote, prevIncome, prevBalance)

		// Fetch headlines for LLM context
		newsItems, _ := finnhub.FetchCompanyNews(ctx, ticker, time.Now().Add(-7*24*time.Hour), time.Now())
		headlines := make([]string, 0, len(newsItems))
		for _, n := range newsItems {
			headlines = append(headlines, n.Headline)
		}

		metricsJSON, _ := json.Marshal(sc)
		insight, err := gemini.AnalyzeFundamentals(ctx, ticker, string(metricsJSON), headlines)
		if err != nil {
			slog.Warn("fundamentals: LLM failed", "ticker", ticker, "err", err)
			// Persist without insight rather than skipping entirely
		}

		flagsJSON, _ := json.Marshal(sc.Flags)
		groupStatus, _ := json.Marshal(map[string]string{"piotroski": sc.PiotroskiGrade})
		insightStr := ""
		if insight != nil {
			b, _ := json.Marshal(insight)
			insightStr = string(b)
		}
		periodEnd, _ := time.Parse("2006-01-02", sc.PeriodEnd)
		if e := db.UpsertScorecard(ctx, ticker, periodEnd, metricsJSON, flagsJSON, groupStatus, insightStr); e != nil {
			slog.Warn("fundamentals: db upsert failed", "ticker", ticker, "err", e)
			failed++
			continue
		}

		slog.Info("fundamentals: refreshed", "ticker", ticker,
			"piotroski", sc.PiotroskiScore, "grade", sc.PiotroskiGrade)
		refreshed++
	}

	slog.Info("fundamentals: done", "refreshed", refreshed, "skipped_fresh", skipped, "failed", failed)
}

// ── market-notify ─────────────────────────────────────────────────────────────

func runMarketNotify(ctx context.Context, db *store.DB, bot *telegram.Bot, finnhub *news.FinnhubClient) {
	chatIDs, err := db.GetLinkedChatIDs(ctx)
	if err != nil {
		slog.Error("market-notify: get chat IDs failed", "err", err)
		return
	}
	if len(chatIDs) == 0 {
		slog.Info("market-notify: no linked users, skipping")
		return
	}

	var globalItems, cryptoItems []news.Item
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		items, e := finnhub.FetchMarketNews(ctx, "general")
		if e != nil {
			slog.Warn("market-notify: fetch general failed", "err", e)
			return
		}
		globalItems = items
	}()
	go func() {
		defer wg.Done()
		items, e := finnhub.FetchMarketNews(ctx, "crypto")
		if e != nil {
			slog.Warn("market-notify: fetch crypto failed", "err", e)
			return
		}
		cryptoItems = items
	}()
	wg.Wait()

	if len(globalItems) == 0 && len(cryptoItems) == 0 {
		slog.Info("market-notify: no news available, skipping")
		return
	}

	if len(globalItems) > 5 {
		globalItems = globalItems[:5]
	}
	if len(cryptoItems) > 3 {
		cryptoItems = cryptoItems[:3]
	}

	msg := buildMarketMessage(globalItems, cryptoItems)
	for _, chatID := range chatIDs {
		if err := bot.SendMessage(ctx, chatID, msg); err != nil {
			slog.Warn("market-notify: send failed", "chat_id", chatID, "err", err)
		}
	}
	slog.Info("market-notify: done", "users", len(chatIDs), "global", len(globalItems), "crypto", len(cryptoItems))
}

func buildMarketMessage(global, crypto []news.Item) string {
	var sb strings.Builder
	sb.WriteString("🌍 <b>Global Update Hari Ini</b>\n\n")

	if len(global) > 0 {
		sb.WriteString("📰 <b>Ekonomi Global</b>\n")
		for _, item := range global {
			sb.WriteString(fmt.Sprintf("• <i>%s</i>\n  🔗 %s\n\n", escapeHTML(item.Headline), item.URL))
		}
	}

	if len(crypto) > 0 {
		sb.WriteString("₿ <b>Bitcoin &amp; Kripto</b>\n")
		for _, item := range crypto {
			sb.WriteString(fmt.Sprintf("• <i>%s</i>\n  🔗 %s\n\n", escapeHTML(item.Headline), item.URL))
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ── sector-notify ─────────────────────────────────────────────────────────────

// sectorETFs maps ticker → nama sektor yang mudah dibaca.
var sectorETFs = []struct {
	Ticker string
	Name   string
}{
	{"XLK", "Technology"},
	{"XLV", "Health Care"},
	{"XLF", "Financials"},
	{"XLY", "Consumer Disc."},
	{"XLP", "Consumer Staples"},
	{"XLE", "Energy"},
	{"XLI", "Industrials"},
	{"XLB", "Materials"},
	{"XLRE", "Real Estate"},
	{"XLC", "Communication"},
	{"XLU", "Utilities"},
}

func runSectorNotify(ctx context.Context, db *store.DB, bot *telegram.Bot, finnhub *news.FinnhubClient) {
	chatIDs, err := db.GetLinkedChatIDs(ctx)
	if err != nil || len(chatIDs) == 0 {
		slog.Info("sector-notify: no linked users, skipping")
		return
	}

	// Fetch all ETF quotes concurrently via Finnhub (free, supports ETFs)
	type quoteResult struct {
		idx int
		pct float64
		err error
	}
	ch := make(chan quoteResult, len(sectorETFs))
	for i, s := range sectorETFs {
		go func(idx int, ticker string) {
			q, e := finnhub.GetQuote(ctx, ticker)
			if e != nil {
				ch <- quoteResult{idx: idx, err: e}
				return
			}
			ch <- quoteResult{idx: idx, pct: q.ChangePct}
		}(i, s.Ticker)
	}

	pctByIdx := make(map[int]float64, len(sectorETFs))
	for range sectorETFs {
		r := <-ch
		if r.err != nil {
			slog.Warn("sector-notify: quote fetch failed", "ticker", sectorETFs[r.idx].Ticker, "err", r.err)
			continue
		}
		pctByIdx[r.idx] = r.pct
	}

	entries := make([]sectorEntry, 0, len(sectorETFs))
	for i, s := range sectorETFs {
		if pct, ok := pctByIdx[i]; ok {
			entries = append(entries, sectorEntry{s.Ticker, s.Name, pct})
		}
	}
	if len(entries) == 0 {
		slog.Warn("sector-notify: no quotes fetched, skipping")
		return
	}

	// Sort descending (sektor terbaik di atas)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Pct > entries[j].Pct })

	msg := buildSectorMessage(entries)
	for _, chatID := range chatIDs {
		if err := bot.SendMessage(ctx, chatID, msg); err != nil {
			slog.Warn("sector-notify: send failed", "chat_id", chatID, "err", err)
		}
	}
	slog.Info("sector-notify: done", "users", len(chatIDs), "sectors", len(entries))
}

type sectorEntry struct {
	Ticker string
	Name   string
	Pct    float64
}

func buildSectorMessage(entries []sectorEntry) string {
	var sb strings.Builder
	sb.WriteString("📡 <b>Sektor US — Siapa Naik Siapa Turun</b>\n")
	sb.WriteString("<i>Perubahan harian (sortir terbaik → terburuk)</i>\n\n")

	for _, e := range entries {
		var emoji string
		switch {
		case e.Pct >= 1.0:
			emoji = "🟢"
		case e.Pct >= 0:
			emoji = "🟡"
		case e.Pct >= -1.0:
			emoji = "🟠"
		default:
			emoji = "🔴"
		}
		sign := "+"
		if e.Pct < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %s  <code>%s%.2f%%</code>\n",
			emoji, e.Ticker, e.Name, sign, e.Pct))
	}

	return strings.TrimRight(sb.String(), "\n")
}
