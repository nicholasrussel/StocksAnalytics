package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tripleboys27/stock-dashboard/api/internal/edgar"
	"github.com/tripleboys27/stock-dashboard/api/internal/fmp"
	"github.com/tripleboys27/stock-dashboard/api/internal/llm"
	"github.com/tripleboys27/stock-dashboard/api/internal/news"
	"github.com/tripleboys27/stock-dashboard/api/internal/scorecard"
	"github.com/tripleboys27/stock-dashboard/api/internal/store"
)

type Server struct {
	db          *store.DB
	finnhub     *news.FinnhubClient
	marketaux   *news.MarketauxClient
	gemini      *llm.GeminiClient
	fmp         *fmp.Client
	edgar       *edgar.Client
	botUsername string
}

func NewServer(db *store.DB, finnhub *news.FinnhubClient, marketaux *news.MarketauxClient, gemini *llm.GeminiClient, fmpClient *fmp.Client, edgarClient *edgar.Client, botUsername string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{db: db, finnhub: finnhub, marketaux: marketaux, gemini: gemini, fmp: fmpClient, edgar: edgarClient, botUsername: botUsername}
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	v1 := r.Group("/v1", internalKeyMiddleware())
	{
		v1.GET("/me", s.getMe)

		user := v1.Group("", userMiddleware(db))
		user.GET("/watchlist", s.getWatchlist)
		user.POST("/watchlist", s.addToWatchlist)
		user.DELETE("/watchlist/:ticker", s.removeFromWatchlist)
		user.GET("/stocks/compare", s.compareStocks) // must be before :ticker routes
		user.GET("/stocks/:ticker/news", s.getStockNews)
		user.GET("/stocks/:ticker/analyze", s.getStockAnalysis)
		user.GET("/stocks/:ticker/fundamentals", s.getFundamentals)
		user.GET("/stocks/:ticker/insider-trading", s.getInsiderTrading)
		user.GET("/markets/news", s.getMarketsNews)
		user.GET("/sectors", s.getSectors)
		user.POST("/telegram/link-token", s.createTelegramLinkToken)
	}

	return r
}

func (s *Server) getMe(c *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(c.GetHeader("X-User-Email")))
	name := c.GetHeader("X-User-Name")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing X-User-Email"})
		return
	}
	user, err := s.db.GetOrCreateUser(c.Request.Context(), email, name)
	if errors.Is(err, store.ErrNotAllowed) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) getWatchlist(c *gin.Context) {
	user := c.MustGet("user").(*store.User)
	items, err := s.db.GetWatchlist(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) addToWatchlist(c *gin.Context) {
	user := c.MustGet("user").(*store.User)
	var body struct {
		Ticker string `json:"ticker" binding:"required"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Status == "" {
		body.Status = "watching"
	}
	if body.Name == "" {
		body.Name = body.Ticker
	}
	item, err := s.db.AddToWatchlist(c.Request.Context(), user.ID, body.Ticker, body.Name, body.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (s *Server) removeFromWatchlist(c *gin.Context) {
	user := c.MustGet("user").(*store.User)
	if err := s.db.RemoveFromWatchlist(c.Request.Context(), user.ID, c.Param("ticker")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) getStockNews(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	ctx := c.Request.Context()

	var (
		mu       sync.Mutex
		combined []news.Item
		wg       sync.WaitGroup
	)

	for _, fn := range []func() ([]news.Item, error){
		func() ([]news.Item, error) { return s.finnhub.FetchCompanyNews(ctx, ticker, from, to) },
		func() ([]news.Item, error) { return s.marketaux.FetchStockNews(ctx, ticker) },
	} {
		fn := fn
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, _ := fn()
			if len(items) > 0 {
				mu.Lock()
				combined = append(combined, items...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	sort.Slice(combined, func(i, j int) bool {
		return combined[i].PublishedAt.After(combined[j].PublishedAt)
	})
	if combined == nil {
		combined = []news.Item{}
	}
	c.JSON(http.StatusOK, combined)
}

func (s *Server) getStockAnalysis(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	to := time.Now()
	from := to.AddDate(0, 0, -7)
	ctx := c.Request.Context()

	var (
		mu       sync.Mutex
		combined []news.Item
		wg       sync.WaitGroup
	)
	for _, fn := range []func() ([]news.Item, error){
		func() ([]news.Item, error) { return s.finnhub.FetchCompanyNews(ctx, ticker, from, to) },
		func() ([]news.Item, error) { return s.marketaux.FetchStockNews(ctx, ticker) },
	} {
		fn := fn
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, _ := fn()
			if len(items) > 0 {
				mu.Lock()
				combined = append(combined, items...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	sort.Slice(combined, func(i, j int) bool {
		return combined[i].PublishedAt.After(combined[j].PublishedAt)
	})

	analysis, err := s.gemini.AnalyzeStockNews(ctx, ticker, combined)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, analysis)
}

func (s *Server) createTelegramLinkToken(c *gin.Context) {
	user := c.MustGet("user").(*store.User)
	if user.TelegramChatID != nil {
		c.JSON(http.StatusOK, gin.H{"already_linked": true})
		return
	}
	token, err := s.db.CreateLinkToken(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	link := "https://t.me/" + s.botUsername + "?start=" + token
	c.JSON(http.StatusCreated, gin.H{"token": token, "link": link})
}

func (s *Server) getMarketsNews(c *gin.Context) {
	ctx := c.Request.Context()

	var (
		mu         sync.Mutex
		globalNews []news.Item
		btcNews    []news.Item
		wg         sync.WaitGroup
	)

	type result struct {
		items []news.Item
		dest  *[]news.Item
	}

	add := func(dest *[]news.Item, fn func() ([]news.Item, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, _ := fn()
			if len(items) > 0 {
				mu.Lock()
				*dest = append(*dest, items...)
				mu.Unlock()
			}
		}()
	}

	add(&globalNews, func() ([]news.Item, error) { return s.finnhub.FetchMarketNews(ctx, "general") })
	add(&btcNews, func() ([]news.Item, error) { return s.finnhub.FetchMarketNews(ctx, "crypto") })
	add(&btcNews, func() ([]news.Item, error) { return s.marketaux.FetchCryptoNews(ctx) })
	wg.Wait()

	// Limit and sort
	sort.Slice(globalNews, func(i, j int) bool { return globalNews[i].PublishedAt.After(globalNews[j].PublishedAt) })
	sort.Slice(btcNews, func(i, j int) bool { return btcNews[i].PublishedAt.After(btcNews[j].PublishedAt) })
	if len(globalNews) > 30 {
		globalNews = globalNews[:30]
	}
	if len(btcNews) > 20 {
		btcNews = btcNews[:20]
	}
	if globalNews == nil {
		globalNews = []news.Item{}
	}
	if btcNews == nil {
		btcNews = []news.Item{}
	}

	c.JSON(http.StatusOK, gin.H{"global": globalNews, "btc": btcNews})
}

// ── /sectors ──────────────────────────────────────────────────────────────────

var sectorETFMeta = []struct {
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

type SectorData struct {
	Ticker string  `json:"ticker"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Change float64 `json:"change_pct"`
}

func (s *Server) getSectors(c *gin.Context) {
	ctx := c.Request.Context()

	nameMap := make(map[string]string, len(sectorETFMeta))
	for _, m := range sectorETFMeta {
		nameMap[m.Ticker] = m.Name
	}

	// Fetch all ETF quotes concurrently via Finnhub (free, supports ETFs)
	type result struct {
		ticker string
		price  float64
		change float64
		err    error
	}
	results := make([]result, len(sectorETFMeta))
	var wg sync.WaitGroup
	for i, m := range sectorETFMeta {
		wg.Add(1)
		go func(idx int, ticker string) {
			defer wg.Done()
			q, err := s.finnhub.GetQuote(ctx, ticker)
			if err != nil {
				results[idx] = result{ticker: ticker, err: err}
				return
			}
			results[idx] = result{ticker: ticker, price: q.Price, change: q.ChangePct}
		}(i, m.Ticker)
	}
	wg.Wait()

	sectors := make([]SectorData, 0, len(sectorETFMeta))
	for _, r := range results {
		if r.err != nil {
			continue
		}
		sectors = append(sectors, SectorData{
			Ticker: r.ticker,
			Name:   nameMap[r.ticker],
			Price:  r.price,
			Change: r.change,
		})
	}
	if len(sectors) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sector quotes"})
		return
	}
	sort.Slice(sectors, func(i, j int) bool { return sectors[i].Change > sectors[j].Change })

	// Build plain-text summary for LLM
	var sb strings.Builder
	for _, sd := range sectors {
		sign := "+"
		if sd.Change < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("%s (%s): %s%.2f%%\n", sd.Ticker, sd.Name, sign, sd.Change))
	}

	var insight interface{}
	if s.gemini != nil {
		if ins, err := s.gemini.AnalyzeSectors(ctx, sb.String()); err == nil {
			insight = ins
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sectors": sectors,
		"insight": insight,
	})
}

func (s *Server) getFundamentals(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	ctx := c.Request.Context()

	// Return cached scorecard if fresh (< 24h)
	cached, err := s.db.GetLatestScorecard(ctx, ticker)
	if err == nil && time.Since(cached.ComputedAt) < 24*time.Hour {
		resp := gin.H{
			"scorecard": json.RawMessage(cached.Metrics),
			"insight":   nil,
			"cached":    true,
		}
		if cached.LLMSummary != nil {
			resp["insight"] = json.RawMessage(*cached.LLMSummary)
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	if s.fmp == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "financial data API not configured"})
		return
	}

	// Fetch latest 2 annual periods concurrently
	var (
		incomes  []fmp.IncomeStatement
		balances []fmp.BalanceSheet
		cashflows []fmp.CashFlowStatement
		quote    *fmp.Quote
		fetchErr error
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	fetch := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				mu.Lock()
				fetchErr = err
				mu.Unlock()
			}
		}()
	}

	fetch(func() error {
		v, err := s.fmp.GetIncomeStatements(ctx, ticker, 2)
		if err == nil {
			mu.Lock(); incomes = v; mu.Unlock()
		}
		return err
	})
	fetch(func() error {
		v, err := s.fmp.GetBalanceSheets(ctx, ticker, 2)
		if err == nil {
			mu.Lock(); balances = v; mu.Unlock()
		}
		return err
	})
	fetch(func() error {
		v, err := s.fmp.GetCashFlowStatements(ctx, ticker, 2)
		if err == nil {
			mu.Lock(); cashflows = v; mu.Unlock()
		}
		return err
	})
	fetch(func() error {
		v, err := s.fmp.GetQuote(ctx, ticker)
		if err == nil {
			mu.Lock(); quote = v; mu.Unlock()
		}
		return err
	})
	wg.Wait()

	if fetchErr != nil || len(incomes) == 0 || len(balances) == 0 || len(cashflows) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch financial data"})
		return
	}

	// Compute scorecard (current period + previous for YoY)
	var prevIncome *fmp.IncomeStatement
	var prevBalance *fmp.BalanceSheet
	if len(incomes) > 1 {
		prevIncome = &incomes[1]
	}
	if len(balances) > 1 {
		prevBalance = &balances[1]
	}

	sc := scorecard.Compute(ticker, incomes[0], balances[0], cashflows[0], quote, prevIncome, prevBalance)

	// Fetch recent news headlines for LLM context
	newsItems, _ := s.finnhub.FetchCompanyNews(ctx, ticker, time.Now().Add(-7*24*time.Hour), time.Now())
	headlines := make([]string, 0, len(newsItems))
	for _, n := range newsItems {
		headlines = append(headlines, n.Headline)
	}

	// LLM insight
	metricsJSON, _ := json.Marshal(sc)
	insight, err := s.gemini.AnalyzeFundamentals(ctx, ticker, string(metricsJSON), headlines)
	if err != nil {
		// Return scorecard without LLM insight rather than failing
		c.JSON(http.StatusOK, gin.H{"scorecard": sc, "insight": nil, "cached": false})
		return
	}

	// Persist to DB
	flagsJSON, _ := json.Marshal(sc.Flags)
	groupStatus, _ := json.Marshal(gin.H{"piotroski": sc.PiotroskiGrade})
	insightJSON, _ := json.Marshal(insight)
	periodEnd, _ := time.Parse("2006-01-02", sc.PeriodEnd)
	_ = s.db.UpsertScorecard(ctx, ticker, periodEnd, metricsJSON, flagsJSON, groupStatus, string(insightJSON))

	c.JSON(http.StatusOK, gin.H{"scorecard": sc, "insight": insight, "cached": false})
}

func (s *Server) compareStocks(c *gin.Context) {
	raw := c.Query("tickers")
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tickers required"})
		return
	}
	var tickers []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t != "" {
			tickers = append(tickers, t)
		}
	}
	if len(tickers) > 5 {
		tickers = tickers[:5]
	}

	type Entry struct {
		Ticker     string          `json:"ticker"`
		PeriodEnd  string          `json:"period_end,omitempty"`
		ComputedAt string          `json:"computed_at,omitempty"`
		Scorecard  json.RawMessage `json:"scorecard,omitempty"`
	}

	ctx := c.Request.Context()
	results := make([]Entry, 0, len(tickers))
	for _, ticker := range tickers {
		row, err := s.db.GetLatestScorecard(ctx, ticker)
		if err != nil {
			results = append(results, Entry{Ticker: ticker})
			continue
		}
		results = append(results, Entry{
			Ticker:     ticker,
			PeriodEnd:  row.PeriodEnd.Format("2006-01-02"),
			ComputedAt: row.ComputedAt.Format(time.RFC3339),
			Scorecard:  row.Metrics,
		})
	}
	c.JSON(http.StatusOK, results)
}

func (s *Server) getInsiderTrading(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	// Fetch up to 12 Form 4 filings from SEC EDGAR; each may contain multiple transactions
	txns, err := s.edgar.GetInsiderTransactions(c.Request.Context(), ticker, 12)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if txns == nil {
		txns = []edgar.InsiderTransaction{}
	}
	c.JSON(http.StatusOK, txns)
}
