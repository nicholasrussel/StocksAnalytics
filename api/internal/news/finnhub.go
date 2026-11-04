package news

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Item struct {
	Headline    string    `json:"headline"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	Summary     string    `json:"summary"`
	Image       string    `json:"image"`
	Category    string    `json:"category"`
	PublishedAt time.Time `json:"published_at"`
}

// Snippet returns the summary cut to at most max runes, so multi-byte
// characters are never split.
func (i Item) Snippet(max int) string {
	r := []rune(strings.TrimSpace(i.Summary))
	if len(r) <= max {
		return string(r)
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

type FinnhubClient struct {
	apiKey string
	http   *http.Client
}

func NewFinnhubClient(apiKey string) *FinnhubClient {
	return &FinnhubClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *FinnhubClient) FetchMarketNews(ctx context.Context, category string) ([]Item, error) {
	url := fmt.Sprintf("https://finnhub.io/api/v1/news?category=%s&token=%s", category, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub: status %d", resp.StatusCode)
	}
	var raw []struct {
		Datetime int64  `json:"datetime"`
		Headline string `json:"headline"`
		Source   string `json:"source"`
		Summary  string `json:"summary"`
		Image    string `json:"image"`
		Category string `json:"category"`
		URL      string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(raw))
	for _, r := range raw {
		if r.URL == "" || r.Headline == "" {
			continue
		}
		items = append(items, Item{
			Headline:    r.Headline,
			Source:      r.Source,
			URL:         r.URL,
			Summary:     r.Summary,
			Image:       r.Image,
			Category:    r.Category,
			PublishedAt: time.Unix(r.Datetime, 0),
		})
	}
	return items, nil
}

// ETFQuote holds the latest price and daily change for a symbol.
type ETFQuote struct {
	Symbol    string
	Price     float64 `json:"c"`  // current price
	Change    float64 `json:"d"`  // change (absolute)
	ChangePct float64 `json:"dp"` // change percent
}

// GetQuote fetches the real-time quote for a single symbol (stock or ETF) from Finnhub.
func (c *FinnhubClient) GetQuote(ctx context.Context, symbol string) (*ETFQuote, error) {
	url := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", symbol, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub: quote %s status %d", symbol, resp.StatusCode)
	}
	var q ETFQuote
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, err
	}
	if q.Price == 0 {
		return nil, fmt.Errorf("finnhub: no quote data for %s", symbol)
	}
	q.Symbol = symbol
	return &q, nil
}

func (c *FinnhubClient) FetchCompanyNews(ctx context.Context, ticker string, from, to time.Time) ([]Item, error) {
	url := fmt.Sprintf(
		"https://finnhub.io/api/v1/company-news?symbol=%s&from=%s&to=%s&token=%s",
		ticker,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		c.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub: status %d", resp.StatusCode)
	}

	var raw []struct {
		Datetime int64  `json:"datetime"`
		Headline string `json:"headline"`
		Source   string `json:"source"`
		Summary  string `json:"summary"`
		Image    string `json:"image"`
		Category string `json:"category"`
		URL      string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(raw))
	for _, r := range raw {
		if r.URL == "" || r.Headline == "" {
			continue
		}
		items = append(items, Item{
			Headline:    r.Headline,
			Source:      r.Source,
			URL:         r.URL,
			Summary:     r.Summary,
			Image:       r.Image,
			Category:    r.Category,
			PublishedAt: time.Unix(r.Datetime, 0),
		})
	}
	return items, nil
}
