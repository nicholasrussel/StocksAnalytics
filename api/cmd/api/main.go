package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/tripleboys27/stock-dashboard/api/internal/api"
	"github.com/tripleboys27/stock-dashboard/api/internal/edgar"
	"github.com/tripleboys27/stock-dashboard/api/internal/fmp"
	"github.com/tripleboys27/stock-dashboard/api/internal/llm"
	"github.com/tripleboys27/stock-dashboard/api/internal/news"
	"github.com/tripleboys27/stock-dashboard/api/internal/store"
	"github.com/tripleboys27/stock-dashboard/api/internal/telegram"
)

func main() {
	_ = godotenv.Load()

	if strings.TrimSpace(os.Getenv("INTERNAL_API_KEY")) == "" {
		slog.Error("INTERNAL_API_KEY is required")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	db, err := store.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	finnhub := news.NewFinnhubClient(os.Getenv("FINNHUB_API_KEY"))
	marketaux := news.NewMarketauxClient(os.Getenv("MARKETAUX_API_KEY"))
	gemini := llm.NewGeminiClient(os.Getenv("GEMINI_API_KEY"), os.Getenv("LLM_MODEL"))
	var fmpClient *fmp.Client
	if key := os.Getenv("FMP_API_KEY"); key != "" {
		fmpClient = fmp.NewClient(key)
	}

	secAgent := os.Getenv("SEC_USER_AGENT")
	if secAgent == "" {
		secAgent = "StockDash contact@example.com"
	}
	edgarClient := edgar.NewClient(secAgent)

	var bot *telegram.Bot
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		bot = telegram.NewBot(token)
	}

	r := api.NewServer(db, finnhub, marketaux, gemini, fmpClient, edgarClient, os.Getenv("TELEGRAM_BOT_USERNAME"), bot, os.Getenv("TELEGRAM_WEBHOOK_SECRET"))
	slog.Info("starting api server", "port", port)
	if err := r.Run(":" + port); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
