-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks 
ADD COLUMN requires_confirmation boolean DEFAULT true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks 
DROP COLUMN IF EXISTS requires_confirmation;
-- +goose StatementEnd

