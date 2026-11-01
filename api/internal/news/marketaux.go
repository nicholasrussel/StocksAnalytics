package news

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type MarketauxClient struct {
	apiKey string
	http   *http.Client
}

func NewMarketauxClient(apiKey string) *MarketauxClient {
	return &MarketauxClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *MarketauxClient) fetchNews(ctx context.Context, params url.Values) ([]Item, error) {
	params.Set("api_token", c.apiKey)
	params.Set("language", "en")
	u := "https://api.marketaux.com/v1/news/all?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketaux: status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Title       string `json:"title"`
			Snippet     string `json:"snippet"`
			URL         string `json:"url"`
			ImageURL    string `json:"image_url"`
			Source      string `json:"source"`
			PublishedAt string `json:"published_at"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(result.Data))
	for _, d := range result.Data {
		if d.URL == "" || d.Title == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, d.PublishedAt)
		if err != nil {
			t = time.Now()
		}
		items = append(items, Item{
			Headline:    d.Title,
			Source:      d.Source,
			URL:         d.URL,
			Summary:     d.Snippet,
			Image:       d.ImageURL,
			PublishedAt: t,
		})
	}
	return items, nil
}

func (c *MarketauxClient) FetchStockNews(ctx context.Context, ticker string) ([]Item, error) {
	return c.fetchNews(ctx, url.Values{
		"symbols":          {ticker},
		"filter_entities":  {"true"},
		"sort":             {"published_at"},
	})
}

func (c *MarketauxClient) FetchCryptoNews(ctx context.Context) ([]Item, error) {
	return c.fetchNews(ctx, url.Values{
		"symbols": {"BTC/USD"},
		"sort":    {"published_at"},
	})
}
