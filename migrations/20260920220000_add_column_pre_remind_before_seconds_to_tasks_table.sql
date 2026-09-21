-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN pre_remind_before_seconds BIGINT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
DROP COLUMN IF EXISTS pre_remind_before_seconds;
-- +goose StatementEnd
