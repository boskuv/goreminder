-- +goose Up
-- +goose StatementBegin
CREATE TABLE targets (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    messenger_related_user_id INTEGER REFERENCES user_messengers(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_targets_user_id ON targets(user_id);
CREATE INDEX idx_targets_messenger_related_user_id ON targets(messenger_related_user_id);
CREATE INDEX idx_targets_created_at ON targets(created_at DESC);
CREATE INDEX idx_targets_deleted_at ON targets(deleted_at) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_targets_deleted_at;
DROP INDEX IF EXISTS idx_targets_created_at;
DROP INDEX IF EXISTS idx_targets_messenger_related_user_id;
DROP INDEX IF EXISTS idx_targets_user_id;
DROP TABLE IF EXISTS targets;
-- +goose StatementEnd

