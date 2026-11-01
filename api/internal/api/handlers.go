package api

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tripleboys27/stock-dashboard/api/internal/llm"
	"github.com/tripleboys27/stock-dashboard/api/internal/news"
	"github.com/tripleboys27/stock-dashboard/api/internal/store"
)

type Server struct {
	db          *store.DB
	finnhub     *news.FinnhubClient
	marketaux   *news.MarketauxClient
	gemini      *llm.GeminiClient
	botUsername string
}

func NewServer(db *store.DB, finnhub *news.FinnhubClient, marketaux *news.MarketauxClient, gemini *llm.GeminiClient, botUsername string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{db: db, finnhub: finnhub, marketaux: marketaux, gemini: gemini, botUsername: botUsername}
	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	v1 := r.Group("/v1", internalKeyMiddleware())
	{
		v1.GET("/me", s.getMe)

		user := v1.Group("", userMiddleware(db))
		user.GET("/watchlist", s.getWatchlist)
		user.POST("/watchlist", s.addToWatchlist)
		user.DELETE("/watchlist/:ticker", s.removeFromWatchlist)
		user.GET("/stocks/:ticker/news", s.getStockNews)
		user.GET("/stocks/:ticker/analyze", s.getStockAnalysis)
		user.GET("/markets/news", s.getMarketsNews)
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
