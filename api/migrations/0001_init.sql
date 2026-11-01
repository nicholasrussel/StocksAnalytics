-- +goose Up
-- +goose StatementBegin

-- ========== AKSES ==========
CREATE TABLE allowed_emails (
  email      TEXT PRIMARY KEY CHECK (email = lower(email)),
  role       TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin','member')),
  is_active  BOOLEAN NOT NULL DEFAULT true,
  added_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email             TEXT NOT NULL UNIQUE REFERENCES allowed_emails(email) ON UPDATE CASCADE,
  name              TEXT,
  notify_channel    TEXT NOT NULL DEFAULT 'telegram' CHECK (notify_channel IN ('telegram')),
  telegram_chat_id  BIGINT,
  min_importance    SMALLINT NOT NULL DEFAULT 4 CHECK (min_importance BETWEEN 1 AND 5),
  digest_enabled    BOOLEAN NOT NULL DEFAULT true,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at     TIMESTAMPTZ
);

CREATE TABLE telegram_link_tokens (
  token       TEXT PRIMARY KEY,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at  TIMESTAMPTZ NOT NULL,
  used_at     TIMESTAMPTZ
);

-- ========== PERUSAHAAN DAN WATCHLIST ==========
CREATE TABLE companies (
  ticker                  TEXT PRIMARY KEY,
  cik                     BIGINT NOT NULL UNIQUE,
  name                    TEXT NOT NULL,
  exchange                TEXT,
  last_filing_seen        TEXT,
  last_fundamentals_accn  TEXT,
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE watchlist_items (
  user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  ticker    TEXT NOT NULL REFERENCES companies(ticker),
  status    TEXT NOT NULL DEFAULT 'watching' CHECK (status IN ('holding','watching')),
  note      TEXT,
  keywords  TEXT[] NOT NULL DEFAULT '{}',
  added_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, ticker)
);

-- ========== BERITA DAN FILING ==========
CREATE TABLE news_items (
  id            BIGSERIAL PRIMARY KEY,
  ticker        TEXT NOT NULL REFERENCES companies(ticker),
  dedup_hash    TEXT NOT NULL,
  source        TEXT NOT NULL CHECK (source IN ('finnhub','marketaux','yahoo_rss','google_rss')),
  publisher     TEXT,
  title         TEXT NOT NULL,
  url           TEXT NOT NULL,
  snippet       TEXT,
  published_at  TIMESTAMPTZ NOT NULL,
  fetched_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  analyzed_at   TIMESTAMPTZ,
  is_relevant   BOOLEAN,
  category      TEXT,
  importance    SMALLINT CHECK (importance BETWEEN 1 AND 5),
  sentiment     TEXT CHECK (sentiment IN ('positif','netral','negatif')),
  summary       TEXT,
  UNIQUE (ticker, dedup_hash)
);

CREATE INDEX news_feed_idx      ON news_items (ticker, published_at DESC);
CREATE INDEX news_queue_idx     ON news_items (fetched_at) WHERE analyzed_at IS NULL;
CREATE INDEX news_important_idx ON news_items (published_at DESC) WHERE importance >= 3;

CREATE TABLE filings (
  accession_no  TEXT PRIMARY KEY,
  ticker        TEXT NOT NULL REFERENCES companies(ticker),
  form          TEXT NOT NULL,
  filed_at      TIMESTAMPTZ NOT NULL,
  report_date   DATE,
  items         TEXT[] NOT NULL DEFAULT '{}',
  doc_url       TEXT NOT NULL,
  importance    SMALLINT CHECK (importance BETWEEN 1 AND 5),
  summary       TEXT,
  analyzed_at   TIMESTAMPTZ
);

CREATE INDEX filings_ticker_idx ON filings (ticker, filed_at DESC);

-- ========== LAPORAN KEUANGAN ==========
CREATE TABLE fundamentals (
  ticker          TEXT NOT NULL REFERENCES companies(ticker),
  fiscal_year     INT  NOT NULL,
  fiscal_period   TEXT NOT NULL CHECK (fiscal_period IN ('Q1','Q2','Q3','Q4','FY')),
  metric          TEXT NOT NULL,
  period_end      DATE NOT NULL,
  value           NUMERIC(24,4) NOT NULL,
  unit            TEXT NOT NULL DEFAULT 'USD',
  source_concept  TEXT NOT NULL,
  accession_no    TEXT NOT NULL,
  is_derived      BOOLEAN NOT NULL DEFAULT false,
  PRIMARY KEY (ticker, fiscal_year, fiscal_period, metric)
);

CREATE TABLE scorecards (
  ticker        TEXT NOT NULL REFERENCES companies(ticker),
  period_end    DATE NOT NULL,
  metrics       JSONB NOT NULL,
  flags         JSONB NOT NULL DEFAULT '[]',
  group_status  JSONB NOT NULL,
  data_quality  TEXT NOT NULL DEFAULT 'ok' CHECK (data_quality IN ('ok','suspect')),
  llm_summary   TEXT,
  computed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (ticker, period_end)
);

-- ========== OPERASIONAL ==========
CREATE TABLE notifications (
  id          BIGSERIAL PRIMARY KEY,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind        TEXT NOT NULL CHECK (kind IN ('news_alert','filing_alert','digest','scorecard')),
  dedupe_key  TEXT NOT NULL,
  channel     TEXT NOT NULL DEFAULT 'telegram',
  status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
  attempts    SMALLINT NOT NULL DEFAULT 0,
  error       TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  sent_at     TIMESTAMPTZ,
  UNIQUE (user_id, dedupe_key)
);

CREATE INDEX notifications_pending_idx ON notifications (created_at) WHERE status = 'pending';

CREATE TABLE source_state (
  source      TEXT NOT NULL,
  ticker      TEXT NOT NULL,
  last_cursor TEXT,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (source, ticker)
);

CREATE TABLE job_runs (
  id           BIGSERIAL PRIMARY KEY,
  job          TEXT NOT NULL,
  started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at  TIMESTAMPTZ,
  status       TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running','ok','partial','failed')),
  stats        JSONB,
  error        TEXT
);

CREATE INDEX job_runs_idx ON job_runs (job, started_at DESC);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS source_state;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS scorecards;
DROP TABLE IF EXISTS fundamentals;
DROP TABLE IF EXISTS filings;
DROP TABLE IF EXISTS news_items;
DROP TABLE IF EXISTS watchlist_items;
DROP TABLE IF EXISTS companies;
DROP TABLE IF EXISTS telegram_link_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS allowed_emails;
