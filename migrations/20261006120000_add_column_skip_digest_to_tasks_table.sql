-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN skip_digest boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
DROP COLUMN IF EXISTS skip_digest;
-- +goose StatementEnd
