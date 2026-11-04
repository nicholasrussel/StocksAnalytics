package main

import (
	"strings"
	"testing"

	"github.com/tripleboys27/stock-dashboard/api/internal/llm"
	"github.com/tripleboys27/stock-dashboard/api/internal/news"
)

func TestBuildMarketMessagesUsesAnalysisAndFallsBack(t *testing.T) {
	global := []news.Item{{Headline: "Fed holds", URL: "https://a.test/?x=1&y=2", Summary: "English text"}}
	crypto := []news.Item{{Headline: "BTC <up>", URL: "https://b.test", Summary: "Source summary"}}
	analyses := []llm.NewsAnalysis{
		{Importance: 4, Sentiment: "negatif", Summary: "Ringkasan a < b"},
		{}, // the model skipped this one
	}

	msgs := buildMarketMessages(global, crypto, analyses)
	if len(msgs) != 1 {
		t.Fatalf("want 1 message, got %d", len(msgs))
	}
	for _, want := range []string{
		"🔴 negatif | Penting: 4/5",
		"Ringkasan a &lt; b",
		"https://a.test/?x=1&amp;y=2",
		"BTC &lt;up&gt;",
		"Source summary",
	} {
		if !strings.Contains(msgs[0], want) {
			t.Errorf("message missing %q:\n%s", want, msgs[0])
		}
	}
	if strings.Contains(msgs[0], "English text") {
		t.Errorf("source summary should be replaced by the analysis:\n%s", msgs[0])
	}
}

func TestBuildMarketMessagesSplitsLongUpdates(t *testing.T) {
	items := make([]news.Item, 8)
	analyses := make([]llm.NewsAnalysis, 8)
	for i := range items {
		items[i] = news.Item{Headline: "h", URL: "https://a.test"}
		analyses[i] = llm.NewsAnalysis{Importance: 3, Sentiment: "netral", Summary: strings.Repeat("x", 900)}
	}

	msgs := buildMarketMessages(items[:5], items[5:], analyses)
	if len(msgs) < 2 {
		t.Fatalf("want a split, got %d message(s)", len(msgs))
	}
	for i, m := range msgs {
		if len(m) > telegramMessageLimit {
			t.Errorf("message %d is %d bytes", i, len(m))
		}
	}
	if got := strings.Count(strings.Join(msgs, ""), "🔗"); got != 8 {
		t.Errorf("want 8 articles across messages, got %d", got)
	}
}
