# GoReminder Telegram bot example

Small [python-telegram-bot](https://github.com/python-telegram-bot/python-telegram-bot) bot that uses
[`../python-client`](../python-client/) against a running GoReminder API.

This is a **sample integration**, not a production bot (no auth beyond Telegram identity, no retries, in-memory session).

## Architecture (this example)

```
Telegram user
    → this bot (commands)
        → GoReminderClient (HTTP /api/v1)
            → GoReminder core
                → PostgreSQL
                → RabbitMQ (optional) → Celery worker → webhook / notifications
```

Linking model (required for queue/done/digest with chat context):

1. Messenger type named `telegram` (created on first `/start` if missing)
2. GoReminder `users` row
3. `messengerRelatedUsers` (MRU): Telegram `user.id` + `chat.id` ↔ internal `user_id`

## Prerequisites

- Running GoReminder API (`GOREMINDER_URL`)
- Bot token from [@BotFather](https://t.me/BotFather)
- Python 3.10+

```bash
cd examples/telegram-bot
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
```

## Run

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
export GOREMINDER_URL=http://localhost:8080   # optional
export GOREMINDER_MESSENGER=telegram          # optional messenger type name

python bot.py
```

Open the bot in Telegram and send `/start`.

## Commands

| Command | API used |
|---------|----------|
| `/start`, `/help` | create/lookup messenger, user, MRU |
| `/tasks` | `GET /users/{id}/tasks` with `messenger_related_user_id`, `status_not=done` |
| `/add <title>` | `POST /tasks` (+ MRU, start in 1 day) |
| `/done <id>` | `POST /tasks/{id}/done` |
| `/mute <id>` / `/unmute <id>` | mute/unmute |
| `/backlog <title>` | `POST /backlogs` |
| `/digest` | `GET /digests` |

## Notes aligned with Swagger

- Creates return `{"id"}` (tasks also `child_id`); deletes are 204.
- Tasks intended for delivery need `messenger_related_user_id` or done/queue may return **422**.
- Prefer Swagger (`/swagger/index.html`) when fields change; keep this bot thin and push logic into the shared client.

## Worker / webhook

Scheduling and message delivery still go through RabbitMQ + Celery worker (see main README). This bot only demonstrates **user-driven HTTP** against the API — it does not replace the worker webhook receiver.
