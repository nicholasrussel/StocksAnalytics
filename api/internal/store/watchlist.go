package store

import (
	"context"
	"strings"
	"time"
)

type WatchlistItem struct {
	Ticker  string    `json:"ticker"`
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Note    *string   `json:"note,omitempty"`
	AddedAt time.Time `json:"added_at"`
}

func (db *DB) GetWatchlist(ctx context.Context, userID string) ([]WatchlistItem, error) {
	rows, err := db.Query(ctx, `
		SELECT w.ticker, c.name, w.status, w.note, w.added_at
		FROM watchlist_items w
		JOIN companies c ON c.ticker = w.ticker
		WHERE w.user_id = $1
		ORDER BY w.added_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var it WatchlistItem
		if err := rows.Scan(&it.Ticker, &it.Name, &it.Status, &it.Note, &it.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []WatchlistItem{}
	}
	return items, rows.Err()
}

func (db *DB) AddToWatchlist(ctx context.Context, userID, ticker, name, status string) (*WatchlistItem, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if err := db.UpsertCompany(ctx, ticker, name); err != nil {
		return nil, err
	}

	var it WatchlistItem
	err := db.QueryRow(ctx, `
		INSERT INTO watchlist_items (user_id, ticker, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, ticker) DO UPDATE
		  SET status = EXCLUDED.status
		RETURNING ticker, $4::text AS name, status, note, added_at
	`, userID, ticker, status, name).Scan(
		&it.Ticker, &it.Name, &it.Status, &it.Note, &it.AddedAt,
	)
	return &it, err
}

func (db *DB) RemoveFromWatchlist(ctx context.Context, userID, ticker string) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	_, err := db.Exec(ctx, `
		DELETE FROM watchlist_items WHERE user_id = $1 AND ticker = $2
	`, userID, ticker)
	return err
}

// GetAllWatchlistTickers returns distinct tickers across all users' watchlists.
func (db *DB) GetAllWatchlistTickers(ctx context.Context) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT ticker FROM watchlist_items ORDER BY ticker`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickers []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tickers = append(tickers, t)
	}
	return tickers, rows.Err()
}
