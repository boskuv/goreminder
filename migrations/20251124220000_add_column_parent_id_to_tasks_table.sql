-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks 
ADD COLUMN parent_id bigint REFERENCES tasks(id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks 
DROP COLUMN IF EXISTS parent_id;
-- +goose StatementEnd

