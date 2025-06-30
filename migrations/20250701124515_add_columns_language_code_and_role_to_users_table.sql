-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN language_code VARCHAR(16);
ALTER TABLE users ADD COLUMN role VARCHAR(32);
COMMENT ON COLUMN users.language_code IS 'IETF BCP 47 language tag, e.g., en, en-US, de';
COMMENT ON COLUMN users.role IS 'Role of the user, e.g., user, admin';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN language_code;
ALTER TABLE users DROP COLUMN role;
-- +goose StatementEnd 