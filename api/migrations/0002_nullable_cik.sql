-- +goose Up
-- +goose StatementBegin
ALTER TABLE companies ALTER COLUMN cik DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE companies SET cik = 0 WHERE cik IS NULL;
ALTER TABLE companies ALTER COLUMN cik SET NOT NULL;
-- +goose StatementEnd
