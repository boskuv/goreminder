"""
Example usage of GoReminder Python Client against a running API.

Requires: GoReminder API at GOREMINDER_URL (default http://localhost:8080).
Creates: user → telegram messenger → MRU → task → backlog → target → digest.
"""

from __future__ import annotations

import json
import os
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

# Allow running from examples/python-client without installing the package.
sys.path.insert(0, str(Path(__file__).resolve().parent))

from requests import HTTPError

from goreminder_client import GoReminderClient


def print_json(data):
    print(json.dumps(data, indent=2, default=str))


def main():
    base_url = os.environ.get("GOREMINDER_URL", "http://localhost:8080")
    client = GoReminderClient(base_url=base_url)

    print("=" * 60)
    print(f"GoReminder API examples → {base_url}")
    print("=" * 60)

    try:
        print("\n0. Healthcheck...")
        print_json(client.healthcheck())
    except Exception as e:
        print(f"API unreachable: {e}")
        return 1

    print("\n1. Creating a user...")
    user = client.create_user(
        name="Example User",
        email="example@goreminder.local",
        timezone="UTC",
        language_code="en",
        role="user",
    )
    print_json(user)
    user_id = user["id"]

    print("\n2. Ensuring messenger type 'telegram'...")
    try:
        messenger = client.get_messenger_id_by_name("telegram")
        messenger_id = messenger["id"]
        print(f"existing messenger id={messenger_id}")
    except HTTPError as e:
        if e.response is not None and e.response.status_code == 404:
            created = client.create_messenger("telegram")
            messenger_id = created["id"]
            print(f"created messenger id={messenger_id}")
        else:
            raise

    print("\n3. Linking messenger-related user (MRU)...")
    mru = client.create_messenger_related_user(
        user_id=user_id,
        messenger_id=messenger_id,
        messenger_user_id="tg-example-1001",
        chat_id="chat-example-1001",
    )
    print_json(mru)
    mru_id = mru["id"]

    print("\n4. Creating a one-time task (with MRU for done/queue)...")
    future = datetime.now(timezone.utc) + timedelta(days=1)
    created_task = client.create_task(
        title="Complete project documentation",
        user_id=user_id,
        start_date=future,
        description="Write API docs",
        messenger_related_user_id=mru_id,
        requires_confirmation=True,
    )
    print_json(created_task)
    task_id = created_task["id"]

    print("\n5. Creating a recurring (cron) parent task...")
    recurring = client.create_task(
        title="Daily standup",
        user_id=user_id,
        start_date=future,
        cron_expression="0 9 * * *",
        requires_confirmation=True,
        messenger_related_user_id=mru_id,
    )
    print_json(recurring)

    print("\n6. Listing user tasks...")
    print_json(client.get_user_tasks(user_id=user_id, page=1, page_size=10))

    print("\n7. Updating task...")
    print_json(
        client.update_task(
            task_id,
            description="Updated description",
            start_date=datetime.now(timezone.utc) + timedelta(days=2),
        )
    )

    print("\n8. Queue schedule (needs MRU on task)...")
    print_json(client.queue_task(task_id, action="schedule", queue_name="celery"))

    print("\n9. Mute / unmute...")
    print_json(client.mute_task(task_id))
    print_json(client.unmute_task(task_id))

    print("\n10. Mark done...")
    print_json(client.mark_task_as_done(task_id))

    print("\n11. Task history...")
    print_json(client.get_task_history(task_id))

    print("\n12. Backlog + batch...")
    print_json(client.create_backlog(title="Implement auth", user_id=user_id, messenger_related_user_id=mru_id))
    print_json(
        client.create_backlogs_batch(
            items="Item A\nItem B\nItem C",
            user_id=user_id,
            separator="\n",
            messenger_related_user_id=mru_id,
        )
    )

    print("\n13. Target...")
    print_json(client.create_target(title="Learn Go", user_id=user_id, messenger_related_user_id=mru_id))

    print("\n14. Digest settings + digest...")
    try:
        print_json(
            client.create_digest_settings(
                user_id=user_id,
                weekday_time="07:00",
                weekend_time="10:00",
                enabled=True,
                messenger_related_user_id=mru_id,
            )
        )
    except HTTPError as e:
        print(f"(settings may already exist) {e}")
    print_json(client.get_digest(user_id=user_id, messenger_related_user_id=mru_id))

    print("\n" + "=" * 60)
    print("Done.")
    print("=" * 60)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
