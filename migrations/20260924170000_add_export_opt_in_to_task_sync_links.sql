-- +goose Up
-- +goose StatementBegin

ALTER TABLE task_sync_links
    ADD COLUMN IF NOT EXISTS export_opt_in BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN task_sync_links.export_opt_in IS
    'True when export was enabled via POST /tasks/{id}/calendar/export (per-task). False for group-scoped export; leaving the binding group then removes the Google event.';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE task_sync_links DROP COLUMN IF EXISTS export_opt_in;

-- +goose StatementEnd
