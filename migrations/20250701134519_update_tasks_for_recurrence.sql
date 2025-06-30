-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
    DROP COLUMN IF EXISTS due_date,
    ADD COLUMN start_date TIMESTAMP,
    ADD COLUMN finish_date TIMESTAMP,
    ADD COLUMN cron_expression VARCHAR(255); 
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
    DROP COLUMN IF EXISTS start_date,
    DROP COLUMN IF EXISTS finish_date,
    DROP COLUMN IF EXISTS cron_expression;
-- +goose StatementEnd