-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN updated_at TIMESTAMP DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks DROP COLUMN updated_at;
-- +goose StatementEnd
