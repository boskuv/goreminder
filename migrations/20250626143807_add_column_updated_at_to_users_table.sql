-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN updated_at TIMESTAMP DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN updated_at;
-- +goose StatementEnd
