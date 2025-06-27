-- +goose Up
-- +goose StatementBegin
-- Make email, password_hash nullable
ALTER TABLE users 
ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users 
ALTER COLUMN password_hash DROP NOT NULL;

UPDATE users SET email = NULL WHERE email = '';
UPDATE users SET password_hash = NULL WHERE password_hash = '';

-- Add a comment explaining the change
COMMENT ON COLUMN users.email IS 'Nullable email for users without email auth (e.g., Telegram bots)';
COMMENT ON COLUMN users.password_hash IS 'Nullable password_hash for users without email auth (e.g., Telegram bots)';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Restore NOT NULL and unique constraint (only for rows where email exists)
UPDATE users SET email = '' WHERE email IS NULL;

ALTER TABLE users 
ALTER COLUMN email SET NOT NULL,
ADD CONSTRAINT users_email_key UNIQUE (email);;
-- +goose StatementEnd
