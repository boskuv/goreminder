-- +goose Up
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT users_email_key;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users USING btree (email);

-- +goose StatementEnd