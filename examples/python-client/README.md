# GoReminder Python Client

Thin `requests`-based client for the GoReminder REST API (`/api/v1`).
Aligned with the OpenAPI/Swagger surface of this repository (`/swagger/index.html`).

## Setup

```bash
cd examples/python-client
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
```

API must be reachable (default `http://localhost:8080`):

```bash
export GOREMINDER_URL=http://localhost:8080
python example_usage.py
```

## Quick start

```python
from datetime import datetime, timedelta, timezone
from goreminder_client import GoReminderClient

client = GoReminderClient(base_url="http://localhost:8080")

user = client.create_user(name="Ada", email="ada@example.com", timezone="UTC")
user_id = user["id"]  # create endpoints return {"id": ...}

# Link Telegram (or another messenger) before queue/done flows that need a chat
messenger_id = client.get_messenger_id_by_name("telegram")["id"]
mru = client.create_messenger_related_user(
    user_id=user_id,
    messenger_id=messenger_id,
    messenger_user_id="123456789",
    chat_id="123456789",
)

task = client.create_task(
    title="Ship docs",
    user_id=user_id,
    start_date=datetime.now(timezone.utc) + timedelta(days=1),
    messenger_related_user_id=mru["id"],
    requires_confirmation=True,
)
# {"id": <parent_or_task>, "child_id": <child_or_0>}
```

## Contract notes (easy to get wrong)

| Topic | Behavior |
|-------|----------|
| **Creates** | Usually `{"id": int}`. Tasks return `{"id", "child_id"}`. |
| **DELETE** | HTTP **204** empty body → client methods return `None`. |
| **Dates** | UTC RFC3339 with `Z` (no fractional seconds in this client). `start_date` on create/update uses `future_date` validation when set. |
| **MRU** | Create requires `user_id`, `messenger_id`, `messenger_user_id`, `chat_id`. Get requires query `chat_id` + `messenger_user_id`. |
| **Queue / done** | Task usually needs `messenger_related_user_id` or queue/done return 422. |
| **Queue body** | `action`, `queue_name`, `task_id` are all required. |
| **Digest** | `GET /digests` uses `user_id` + optional `start_date_from` / `start_date_to` (not a single `date`). |
| **Attachments** | Routes exist always; if `attachments.enabled=false` API returns **503**. |
| **Errors** | `{"error": "..."}`; client raises `requests.HTTPError`. |

Canonical reference: **Swagger UI** on the running server, not this README.

## Method map

### Tasks
`create_task`, `get_task`, `get_all_tasks`, `update_task`, `delete_task`, `mark_task_as_done`, `mute_task`, `unmute_task`, `get_task_history`, `get_user_tasks`, `get_user_task_history`, `queue_task`

### Attachments
`init_attachment`, `upload_attachment_direct`, `complete_attachment`, `list_attachments`, `get_attachment_download_url`, `get_attachment_content`, `delete_attachment`

### Users / messengers
`create_user`, `get_user`, `get_all_users`, `update_user`, `delete_user`  
`create_messenger`, `get_messenger`, `get_messenger_id_by_name`, `get_all_messengers`  
`create_messenger_related_user`, `get_messenger_related_user`, `get_all_messenger_related_users`, `get_user_id_by_messenger_user_id`

### Backlogs / targets / digests
Full CRUD helpers matching `/backlogs`, `/targets`, `/digests` and `/digests/settings`.

### System
`healthcheck`, `version` (root paths, not under `/api/v1`)

## Recurrence

- One-shot: no `cron_expression` / `rrule`
- Cron parent: `cron_expression` + usually `requires_confirmation=true` → may create a child (`child_id`)
- RRULE: pass `rrule` instead of `cron_expression` (API treats them as mutually exclusive at service layer)

See the main project README for mute/unmute and child-task behavior.

## Related example

Telegram bot that uses this client: [`../telegram-bot/`](../telegram-bot/).
