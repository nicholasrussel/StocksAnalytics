package store

import (
	"context"
	"encoding/json"
	"time"
)

type ScorecardRow struct {
	Ticker      string
	PeriodEnd   time.Time
	Metrics     json.RawMessage
	Flags       json.RawMessage
	GroupStatus json.RawMessage
	LLMSummary  *string
	ComputedAt  time.Time
}

// UpsertScorecard saves or updates a scorecard.
// metrics and flags are JSON-encoded values.
func (db *DB) UpsertScorecard(ctx context.Context, ticker string, periodEnd time.Time, metrics, flags, groupStatus json.RawMessage, llmSummary string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO scorecards (ticker, period_end, metrics, flags, group_status, llm_summary, computed_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (ticker, period_end) DO UPDATE
		  SET metrics = EXCLUDED.metrics,
		      flags = EXCLUDED.flags,
		      group_status = EXCLUDED.group_status,
		      llm_summary = EXCLUDED.llm_summary,
		      computed_at = now()
	`, ticker, periodEnd, metrics, flags, groupStatus, llmSummary)
	return err
}

// GetLatestScorecard returns the most recent scorecard for a ticker.
func (db *DB) GetLatestScorecard(ctx context.Context, ticker string) (*ScorecardRow, error) {
	var row ScorecardRow
	err := db.QueryRow(ctx, `
		SELECT ticker, period_end, metrics, flags, group_status, llm_summary, computed_at
		FROM scorecards
		WHERE ticker = $1
		ORDER BY period_end DESC
		LIMIT 1
	`, ticker).Scan(
		&row.Ticker, &row.PeriodEnd, &row.Metrics, &row.Flags,
		&row.GroupStatus, &row.LLMSummary, &row.ComputedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}
