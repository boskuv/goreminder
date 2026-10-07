-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN shift_from_completion boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
DROP COLUMN IF EXISTS shift_from_completion;
-- +goose StatementEnd
