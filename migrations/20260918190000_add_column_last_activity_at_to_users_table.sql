-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN last_activity_at TIMESTAMPTZ DEFAULT NULL;
CREATE INDEX idx_users_last_activity_at ON users (last_activity_at DESC NULLS LAST) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_last_activity_at;
ALTER TABLE users DROP COLUMN last_activity_at;
-- +goose StatementEnd
