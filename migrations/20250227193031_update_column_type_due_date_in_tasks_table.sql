-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ALTER COLUMN due_date TYPE TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks ALTER COLUMN due_date TYPE DATE;
-- +goose StatementEnd
