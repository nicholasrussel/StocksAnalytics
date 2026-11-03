package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tripleboys27/stock-dashboard/api/internal/news"
)

const maxRetries = 3

type GeminiClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-3.1-flash-lite"
	}
	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

type StockAnalysis struct {
	Sentiment  string   `json:"sentiment"`
	Summary    string   `json:"summary"`
	Highlights []string `json:"highlights"`
}

func (c *GeminiClient) AnalyzeStockNews(ctx context.Context, ticker string, items []news.Item) (*StockAnalysis, error) {
	if len(items) == 0 {
		return &StockAnalysis{
			Sentiment:  "netral",
			Summary:    "Tidak ada berita terbaru yang tersedia untuk dianalisis.",
			Highlights: []string{},
		}, nil
	}

	var sb strings.Builder
	limit := len(items)
	if limit > 15 {
		limit = 15
	}
	for _, item := range items[:limit] {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", item.Headline, item.Source))
		if item.Summary != "" {
			snippet := item.Summary
			if len(snippet) > 150 {
				snippet = snippet[:150]
			}
			sb.WriteString(fmt.Sprintf("  %s\n", snippet))
		}
	}

	prompt := fmt.Sprintf(`Kamu adalah analis keuangan yang membantu investor retail Indonesia.
Analisis berita saham %s berikut (7 hari terakhir). Berikan respons dalam JSON:
{"sentiment":"positif|netral|negatif","summary":"ringkasan 3-4 kalimat Bahasa Indonesia tentang kondisi saham dan hal yang perlu diperhatikan investor","highlights":["poin penting 1","poin penting 2","poin penting 3"]}

Berita:
%s`, ticker, sb.String())

	text, err := c.generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var analysis StockAnalysis
	if err := json.Unmarshal([]byte(stripCodeFence(text)), &analysis); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse JSON: %w — raw: %.200s", err, text)
	}
	if analysis.Highlights == nil {
		analysis.Highlights = []string{}
	}
	return &analysis, nil
}

// FundamentalsInsight is the LLM analysis of a company's financial scorecard.
type FundamentalsInsight struct {
	Summary     string   `json:"summary"`     // 3-4 kalimat gambaran umum
	Strengths   []string `json:"strengths"`   // kelebihan utama
	Risks       []string `json:"risks"`       // risiko utama
	Outlook     string   `json:"outlook"`     // "positif" | "netral" | "negatif"
	ContextNote string   `json:"context_note"` // kaitan dengan berita terkini
}

// AnalyzeFundamentals generates LLM insights from a scorecard + recent news headlines.
// metricsJSON is the JSON-encoded Scorecard; newsHeadlines is a list of recent news titles.
func (c *GeminiClient) AnalyzeFundamentals(ctx context.Context, ticker, metricsJSON string, newsHeadlines []string) (*FundamentalsInsight, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Kamu adalah analis fundamental saham. Analisis kondisi keuangan %s berdasarkan data berikut.\n\n", ticker))
	sb.WriteString("=== DATA KEUANGAN ===\n")
	sb.WriteString(metricsJSON)
	sb.WriteString("\n\n")

	if len(newsHeadlines) > 0 {
		sb.WriteString("=== BERITA TERKINI ===\n")
		limit := 10
		if len(newsHeadlines) < limit {
			limit = len(newsHeadlines)
		}
		for _, h := range newsHeadlines[:limit] {
			sb.WriteString("- " + h + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Berikan analisis dalam JSON berikut (jangan tambahkan teks lain):
{
  "summary": "gambaran kondisi keuangan 3-4 kalimat dalam Bahasa Indonesia, sebutkan angka kunci",
  "strengths": ["kelebihan 1", "kelebihan 2", "kelebihan 3"],
  "risks": ["risiko 1", "risiko 2"],
  "outlook": "positif|netral|negatif",
  "context_note": "1-2 kalimat kaitan antara kondisi keuangan dengan berita terkini di atas"
}`)

	text, err := c.generate(ctx, sb.String())
	if err != nil {
		return nil, err
	}

	var insight FundamentalsInsight
	if err := json.Unmarshal([]byte(stripCodeFence(text)), &insight); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse fundamentals insight: %w — raw: %.200s", err, text)
	}
	if insight.Outlook == "" {
		insight.Outlook = "netral"
	}
	return &insight, nil
}

// SectorInsight is the LLM analysis of S&P 500 sector rotation.
type SectorInsight struct {
	Summary   string   `json:"summary"`   // 2-3 kalimat kondisi pasar
	Leaders   []string `json:"leaders"`   // penjelasan sektor pemimpin
	Laggards  []string `json:"laggards"`  // penjelasan sektor terlemah
	Watch     []string `json:"watch"`     // sektor yang perlu dicermati untuk riset ticker
	Sentiment string   `json:"sentiment"` // "positif" | "netral" | "negatif"
}

// AnalyzeSectors generates LLM insight about current sector rotation.
// sectorData is a plain-text list of "TICKER (Name): +/-X.XX%" sorted by performance.
func (c *GeminiClient) AnalyzeSectors(ctx context.Context, sectorData string) (*SectorInsight, error) {
	prompt := fmt.Sprintf(`Kamu adalah analis pasar saham Amerika untuk investor retail Indonesia.
Berikut performa sektor S&P 500 hari ini (sortir terbaik ke terburuk):

%s
Berikan analisis rotasi sektor dalam JSON (jangan tambahkan teks lain):
{
  "summary": "2-3 kalimat kondisi pasar hari ini dalam Bahasa Indonesia",
  "leaders": ["XLE memimpin karena ... (1 kalimat)", "XLI naik karena ..."],
  "laggards": ["XLU tertinggal karena ... (1 kalimat)"],
  "watch": ["sektor yang menarik untuk riset ticker baru, 1 kalimat per item"],
  "sentiment": "positif|netral|negatif"
}
leaders & laggards: 2-3 item masing-masing. watch: 1-2 item.`, sectorData)

	text, err := c.generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var insight SectorInsight
	if err := json.Unmarshal([]byte(stripCodeFence(text)), &insight); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse sector insight: %w — raw: %.200s", err, text)
	}
	if insight.Sentiment == "" {
		insight.Sentiment = "netral"
	}
	if insight.Leaders == nil {
		insight.Leaders = []string{}
	}
	if insight.Laggards == nil {
		insight.Laggards = []string{}
	}
	if insight.Watch == nil {
		insight.Watch = []string{}
	}
	return &insight, nil
}

type NewsAnalysis struct {
	Importance int    `json:"importance"`
	Sentiment  string `json:"sentiment"`
	Summary    string `json:"summary"`
}

// AnalyzeSingleNews scores one news article for importance (1-5), sentiment, and a short summary.
func (c *GeminiClient) AnalyzeSingleNews(ctx context.Context, ticker, title, snippet string) (*NewsAnalysis, error) {
	content := title
	if snippet != "" {
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		content += "\n" + snippet
	}

	prompt := fmt.Sprintf(`Kamu adalah analis keuangan. Nilai berita saham berikut untuk investor Indonesia.

Saham: %s
Berita: %s

Respons JSON (jangan tambahkan teks lain):
{"importance":3,"sentiment":"netral","summary":"ringkasan 1-2 kalimat Bahasa Indonesia"}

importance: 1=tidak relevan, 2=info umum, 3=perlu perhatian, 4=penting, 5=sangat kritis
sentiment: positif|netral|negatif`, ticker, content)

	text, err := c.generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var analysis NewsAnalysis
	if err := json.Unmarshal([]byte(stripCodeFence(text)), &analysis); err != nil {
		return nil, fmt.Errorf("gemini: failed to parse news analysis: %w — raw: %.200s", err, text)
	}
	if analysis.Importance < 1 || analysis.Importance > 5 {
		analysis.Importance = 3
	}
	if analysis.Sentiment == "" {
		analysis.Sentiment = "netral"
	}
	return &analysis, nil
}

// stripCodeFence removes markdown code fences (```json ... ```) from Gemini responses.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	for _, fence := range []string{"```json", "```JSON", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			if idx := strings.LastIndex(s, "```"); idx != -1 {
				s = s[:idx]
			}
			return strings.TrimSpace(s)
		}
	}
	// Fallback: find the first '{' and last '}' to extract bare JSON object
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

func (c *GeminiClient) generate(ctx context.Context, prompt string) (string, error) {
	apiURL := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		c.model,
	)

	body := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.3,
			"maxOutputTokens": 700,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	var (
		resp    *http.Response
		lastErr error
	)
	for attempt := range maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
			// rebuild request body (reader is consumed after first attempt)
			req, err = http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(b))
			if err != nil {
				return "", err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-goog-api-key", c.apiKey)
		}
		resp, err = c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == 429 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("gemini: status %d — %s", resp.StatusCode, string(body))
			slog.Warn("gemini retrying", "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}
		break
	}
	if resp == nil {
		return "", lastErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("gemini error", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("gemini: status %d — %s", resp.StatusCode, string(body))
	}

	var gr struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}
