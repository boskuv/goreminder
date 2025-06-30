-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN timezone VARCHAR(64);
COMMENT ON COLUMN users.timezone IS 'IANA timezone name, e.g., Europe/Berlin, America/New_York';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN timezone;
-- +goose StatementEnd 