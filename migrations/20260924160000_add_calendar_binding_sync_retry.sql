-- +goose Up
-- +goose StatementBegin

ALTER TABLE calendar_bindings
    ADD COLUMN IF NOT EXISTS sync_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_calendar_bindings_error_retry
    ON calendar_bindings(status, next_retry_at)
    WHERE deleted_at IS NULL AND status = 'error';

COMMENT ON COLUMN calendar_bindings.sync_attempts IS
    'Consecutive failed import sync attempts; reset on success. Used for exponential backoff.';
COMMENT ON COLUMN calendar_bindings.next_retry_at IS
    'When status=error, scheduler may retry import sync after this time.';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_calendar_bindings_error_retry;
ALTER TABLE calendar_bindings
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS sync_attempts;

-- +goose StatementEnd
