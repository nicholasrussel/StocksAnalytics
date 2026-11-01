package store

import (
	"context"
	"strings"
	"time"
)

type Company struct {
	Ticker    string    `json:"ticker"`
	Name      string    `json:"name"`
	Exchange  *string   `json:"exchange,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (db *DB) UpsertCompany(ctx context.Context, ticker, name string) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	_, err := db.Exec(ctx, `
		INSERT INTO companies (ticker, name, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (ticker) DO UPDATE
		  SET name       = COALESCE(NULLIF(EXCLUDED.name, ''), companies.name),
		      updated_at = now()
	`, ticker, name)
	return err
}
