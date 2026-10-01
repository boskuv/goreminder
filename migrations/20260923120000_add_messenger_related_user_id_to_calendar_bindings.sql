-- +goose Up
ALTER TABLE calendar_bindings
    ADD COLUMN messenger_related_user_id INTEGER REFERENCES user_messengers(id) ON DELETE SET NULL;

CREATE INDEX idx_calendar_bindings_messenger_related_user_id
    ON calendar_bindings(messenger_related_user_id)
    WHERE messenger_related_user_id IS NOT NULL;

COMMENT ON COLUMN calendar_bindings.messenger_related_user_id IS
    'Optional default messenger for imported tasks; when set, import publishes schedule_task to the worker';

-- +goose Down
DROP INDEX IF EXISTS idx_calendar_bindings_messenger_related_user_id;
ALTER TABLE calendar_bindings DROP COLUMN IF EXISTS messenger_related_user_id;
