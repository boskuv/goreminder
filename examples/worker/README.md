# GoReminder sample worker

Minimal **non-Celery** consumer that matches the GoReminder RabbitMQ contract and
delivers reminders via HTTP webhook (same shape as production `POST /send_message`).

This is a **sample**: Redis ZSET scheduling instead of APScheduler/SQLite used in
the real worker. Digests are acknowledged as no-ops.

## Flow

```
GoReminder API
  → RabbitMQ exchange/queue `celery`
      body: { "task": "worker.schedule_task"|"worker.delete_task", "args": [...] }
  → this worker
      → Redis HASH  goreminder:job:{messenger}_{task_id}
      → Redis ZSET  goreminder:due   score = unix fire time
  → poller
      → POST WEBHOOK_URL   (default http://localhost:8001/send_message)
          → examples/telegram-bot webhook receiver
```

## Contract (must match `pkg/queue`)

**`worker.schedule_task` args** (order fixed):

1. `messenger_name` (string)
2. `chat_id` (string)
3. `task_id` (int)
4. `title` (string)
5. `description` (string)
6. `start_date` (RFC3339 / null — may include fractional seconds)
7. `cron_expression` (string / null) — **not** `rrule`
8. `requires_confirmation` (bool)

**`worker.delete_task` args:**

1. `task_id`
2. `messenger_name`

Job id: `{messenger_name}_{task_id}` (same as production).

**Webhook body** (when due):

```json
{
  "chat_id": "123",
  "text": "🔔 Title",
  "task_id": 42,
  "reply_markup": {
    "inline_keyboard": [[
      {"text": "✅ Выполнено", "callback_data": "done:42"},
      {"text": "⏰ Отложить", "callback_data": "later:42"}
    ]]
  }
}
```

`reply_markup` only when `requires_confirmation` is true.

## Prerequisites

- RabbitMQ (same as `producer.*` in core config)
- Redis
- Webhook receiver — [`../telegram-bot`](../telegram-bot/) with `WEBHOOK_PORT=8001`, or any stub that accepts the JSON above

```bash
cd examples/worker
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
```

## Run

```bash
export BROKER_URL=amqp://guest:guest@localhost:5672/
export REDIS_URL=redis://localhost:6379/0
export WEBHOOK_URL=http://localhost:8001/send_message

python run.py
```

Optional: `QUEUE_NAME`, `EXCHANGE` (defaults `celery`), `POLL_INTERVAL_SEC`, `ANTI_DUPE_TTL`.

## Smoke test without the full stack

Publish a schedule message (exchange `celery`, routing key `celery`):

```bash
# example: fire in ~15s — adjust start_date
python - <<'PY'
import json, pika
from datetime import datetime, timedelta, timezone
body = {
  "task": "worker.schedule_task",
  "args": [
    "telegram", "YOUR_CHAT_ID", 999001, "Worker smoke", "",
    (datetime.now(timezone.utc) + timedelta(seconds=15)).strftime("%Y-%m-%dT%H:%M:%SZ"),
    None, True,
  ],
}
conn = pika.BlockingConnection(pika.URLParameters("amqp://guest:guest@localhost:5672/"))
ch = conn.channel()
ch.basic_publish("celery", "celery", json.dumps(body), pika.BasicProperties(content_type="application/json"))
conn.close()
print("published", body)
PY
```

## Limits (intentionally)

- No Celery app / result backend — plain AMQP + JSON
- Cron via `croniter` only; **RRULE** is not in queue args (resolve via API if needed)
- Digest settings tasks are logged and ignored
- Not multi-replica safe beyond ZREM claim (demo-grade)
- Retries are a simple re-ZADD +30s after webhook failure

Prefer the production worker for real deployments; use this to understand the contract or bootstrap a custom messenger.
