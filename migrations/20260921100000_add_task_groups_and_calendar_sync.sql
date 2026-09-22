-- +goose Up
-- +goose StatementBegin

-- Reminder / task groups (sync scoping layer)
CREATE TABLE task_groups (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_task_groups_user_id ON task_groups(user_id);
CREATE INDEX idx_task_groups_deleted_at ON task_groups(deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE tasks
    ADD COLUMN group_id BIGINT REFERENCES task_groups(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_group_id ON tasks(group_id) WHERE group_id IS NOT NULL;

-- Google OAuth accounts (tokens encrypted at rest by application)
CREATE TABLE google_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    google_sub VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    access_token_enc BYTEA NOT NULL,
    refresh_token_enc BYTEA NOT NULL,
    token_expiry TIMESTAMP NOT NULL,
    scopes TEXT NOT NULL DEFAULT '',
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_google_accounts_google_sub UNIQUE (google_sub),
    CONSTRAINT uq_google_accounts_user_google_sub UNIQUE (user_id, google_sub)
);

CREATE INDEX idx_google_accounts_user_id ON google_accounts(user_id);

-- Per-calendar sync bindings
CREATE TABLE calendar_bindings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    google_account_id BIGINT NOT NULL REFERENCES google_accounts(id) ON DELETE CASCADE,
    google_calendar_id VARCHAR(255) NOT NULL,
    calendar_summary VARCHAR(512),
    direction VARCHAR(16) NOT NULL DEFAULT 'import'
        CHECK (direction IN ('import', 'export', 'both')),
    group_id BIGINT REFERENCES task_groups(id) ON DELETE SET NULL,
    sync_token TEXT,
    last_synced_at TIMESTAMP,
    last_error TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'error', 'disconnected')),
    delete_policy VARCHAR(32) NOT NULL DEFAULT 'soft_delete_imported'
        CHECK (delete_policy IN ('soft_delete_imported', 'mute_imported', 'keep')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT uq_calendar_bindings_account_calendar UNIQUE (google_account_id, google_calendar_id)
);

CREATE INDEX idx_calendar_bindings_user_id ON calendar_bindings(user_id);
CREATE INDEX idx_calendar_bindings_google_account_id ON calendar_bindings(google_account_id);
CREATE INDEX idx_calendar_bindings_group_id ON calendar_bindings(group_id) WHERE group_id IS NOT NULL;
CREATE INDEX idx_calendar_bindings_deleted_at ON calendar_bindings(deleted_at) WHERE deleted_at IS NULL;

-- Task ↔ Google Event link metadata
CREATE TABLE task_sync_links (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    provider VARCHAR(64) NOT NULL DEFAULT 'google_calendar',
    google_calendar_id VARCHAR(255) NOT NULL,
    google_event_id VARCHAR(255) NOT NULL,
    etag VARCHAR(255),
    google_updated_at TIMESTAMP,
    origin VARCHAR(16) NOT NULL
        CHECK (origin IN ('imported', 'exported')),
    sync_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    calendar_binding_id BIGINT REFERENCES calendar_bindings(id) ON DELETE SET NULL,
    duration_seconds INTEGER,
    last_synced_at TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_task_sync_links_task_provider_event UNIQUE (task_id, provider, google_event_id),
    CONSTRAINT uq_task_sync_links_provider_calendar_event UNIQUE (provider, google_calendar_id, google_event_id)
);

CREATE INDEX idx_task_sync_links_task_id ON task_sync_links(task_id);
CREATE INDEX idx_task_sync_links_calendar_binding_id ON task_sync_links(calendar_binding_id);
CREATE INDEX idx_task_sync_links_provider_calendar ON task_sync_links(provider, google_calendar_id);

-- Async sync work queue (export/import side effects)
CREATE TABLE sync_outbox (
    id BIGSERIAL PRIMARY KEY,
    kind VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sync_outbox_status_next_retry ON sync_outbox(status, next_retry_at)
    WHERE status IN ('pending', 'processing');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_sync_outbox_status_next_retry;
DROP TABLE IF EXISTS sync_outbox;

DROP INDEX IF EXISTS idx_task_sync_links_provider_calendar;
DROP INDEX IF EXISTS idx_task_sync_links_calendar_binding_id;
DROP INDEX IF EXISTS idx_task_sync_links_task_id;
DROP TABLE IF EXISTS task_sync_links;

DROP INDEX IF EXISTS idx_calendar_bindings_deleted_at;
DROP INDEX IF EXISTS idx_calendar_bindings_group_id;
DROP INDEX IF EXISTS idx_calendar_bindings_google_account_id;
DROP INDEX IF EXISTS idx_calendar_bindings_user_id;
DROP TABLE IF EXISTS calendar_bindings;

DROP INDEX IF EXISTS idx_google_accounts_user_id;
DROP TABLE IF EXISTS google_accounts;

DROP INDEX IF EXISTS idx_tasks_group_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS group_id;

DROP INDEX IF EXISTS idx_task_groups_deleted_at;
DROP INDEX IF EXISTS idx_task_groups_user_id;
DROP TABLE IF EXISTS task_groups;

-- +goose StatementEnd
