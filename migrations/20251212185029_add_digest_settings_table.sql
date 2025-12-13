-- +goose Up
-- +goose StatementBegin
CREATE TABLE digest_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    messenger_related_user_id INTEGER REFERENCES user_messengers(id) ON DELETE SET NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    weekday_time VARCHAR(5) NOT NULL DEFAULT '07:00',  -- Format: HH:MM
    weekend_time VARCHAR(5) NOT NULL DEFAULT '10:00',  -- Format: HH:MM
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, messenger_related_user_id)
);

CREATE INDEX idx_digest_settings_user_id ON digest_settings(user_id);
CREATE INDEX idx_digest_settings_messenger_related_user_id ON digest_settings(messenger_related_user_id);
CREATE INDEX idx_digest_settings_enabled ON digest_settings(enabled) WHERE enabled = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_digest_settings_enabled;
DROP INDEX IF EXISTS idx_digest_settings_messenger_related_user_id;
DROP INDEX IF EXISTS idx_digest_settings_user_id;
DROP TABLE IF EXISTS digest_settings;
-- +goose StatementEnd

