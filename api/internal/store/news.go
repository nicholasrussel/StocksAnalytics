package store

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

type NewsItem struct {
	ID          int64
	Ticker      string
	Title       string
	URL         string
	Snippet     string
	Source      string
	Publisher   string
	PublishedAt time.Time
}

type NotifyCandidate struct {
	NewsID      int64
	Ticker      string
	Title       string
	URL         string
	Importance  int
	Sentiment   string
	Summary     string
	PublishedAt time.Time
	UserID      string
	ChatID      int64
}

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// UpsertNewsItem inserts a news item; ignores duplicate (same ticker + url).
// Returns true if the row was newly inserted.
func (db *DB) UpsertNewsItem(ctx context.Context, ticker, title, url, snippet, source, publisher string, publishedAt time.Time) (bool, error) {
	hash := md5Hash(url)
	tag, err := db.Exec(ctx, `
		INSERT INTO news_items (ticker, dedup_hash, source, publisher, title, url, snippet, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (ticker, dedup_hash) DO NOTHING
	`, ticker, hash, source, publisher, title, url, snippet, publishedAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// GetUnanalyzedNews returns news from the last 48 hours that haven't been analyzed.
func (db *DB) GetUnanalyzedNews(ctx context.Context) ([]NewsItem, error) {
	rows, err := db.Query(ctx, `
		SELECT id, ticker, title, url, COALESCE(snippet, ''), source, COALESCE(publisher, ''), published_at
		FROM news_items
		WHERE analyzed_at IS NULL
		  AND published_at > now() - interval '48 hours'
		ORDER BY published_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []NewsItem
	for rows.Next() {
		var item NewsItem
		if err := rows.Scan(&item.ID, &item.Ticker, &item.Title, &item.URL, &item.Snippet, &item.Source, &item.Publisher, &item.PublishedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// MarkNewsAnalyzed saves LLM analysis results for a news item.
func (db *DB) MarkNewsAnalyzed(ctx context.Context, id int64, importance int, sentiment, summary string) error {
	_, err := db.Exec(ctx, `
		UPDATE news_items
		SET analyzed_at = now(), importance = $1, sentiment = $2, summary = $3
		WHERE id = $4
	`, importance, sentiment, summary, id)
	return err
}

// GetNotifyCandidates returns analyzed news that users haven't been notified about yet.
// Only returns items published in the last 48 hours where importance >= user's min_importance.
func (db *DB) GetNotifyCandidates(ctx context.Context) ([]NotifyCandidate, error) {
	rows, err := db.Query(ctx, `
		SELECT n.id, n.ticker, n.title, n.url,
		       n.importance, COALESCE(n.sentiment,'netral'), COALESCE(n.summary,''),
		       n.published_at, u.id, u.telegram_chat_id
		FROM news_items n
		JOIN watchlist_items w ON w.ticker = n.ticker
		JOIN users u ON u.id = w.user_id
		WHERE n.analyzed_at IS NOT NULL
		  AND n.importance IS NOT NULL
		  AND n.importance >= u.min_importance
		  AND n.published_at > now() - interval '48 hours'
		  AND u.telegram_chat_id IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM notifications notif
		      WHERE notif.user_id = u.id
		        AND notif.dedupe_key = 'news:' || n.id::text
		  )
		ORDER BY u.id, n.importance DESC, n.published_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []NotifyCandidate
	for rows.Next() {
		var c NotifyCandidate
		if err := rows.Scan(&c.NewsID, &c.Ticker, &c.Title, &c.URL, &c.Importance, &c.Sentiment, &c.Summary, &c.PublishedAt, &c.UserID, &c.ChatID); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// RecordNotificationSent marks a news item as notified for a user so it isn't sent again.
func (db *DB) RecordNotificationSent(ctx context.Context, userID string, newsID int64) error {
	_, err := db.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, dedupe_key, channel, status, sent_at)
		VALUES ($1, 'news_alert', $2, 'telegram', 'sent', now())
		ON CONFLICT (user_id, dedupe_key) DO NOTHING
	`, userID, fmt.Sprintf("news:%d", newsID))
	return err
}
