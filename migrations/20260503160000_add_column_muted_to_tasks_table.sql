-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN muted boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
DROP COLUMN IF EXISTS muted;
-- +goose StatementEnd
