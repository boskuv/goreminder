-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS rrule TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
    DROP COLUMN IF EXISTS rrule;
-- +goose StatementEnd
