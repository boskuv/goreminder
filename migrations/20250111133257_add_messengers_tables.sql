-- +goose Up
-- +goose StatementBegin
CREATE TABLE messengers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_messengers (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    messenger_id INT NOT NULL REFERENCES messengers(id) ON DELETE CASCADE,
    chat_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, messenger_id, chat_id)
);

CREATE TABLE task_messenger_notifications (
    id SERIAL PRIMARY KEY,
    task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_messenger_id INT NOT NULL REFERENCES user_messengers(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE messengers, user_messengers, task_messenger_notifications;
-- +goose StatementEnd
