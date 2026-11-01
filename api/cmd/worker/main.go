package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tripleboys27/stock-dashboard/api/internal/llm"
	"github.com/tripleboys27/stock-dashboard/api/internal/news"
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
