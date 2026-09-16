"""Dispatch Celery-style {task, args} messages into the Redis due-store."""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import Any

from croniter import croniter

from datetime_util import parse_datetime
from store import DueStore, job_id
from webhook import send_reminder

log = logging.getLogger("goreminder-worker.handlers")


def _as_bool(v: Any) -> bool:
    if isinstance(v, bool):
        return v
    if v is None:
        return False
    if isinstance(v, (int, float)):
        return bool(v)
    return str(v).strip().lower() in ("1", "true", "yes", "y")


def _next_cron(cron_expression: str, after: datetime) -> datetime:
    """Next fire time. GoReminder uses 5-field cron; croniter accepts that."""
    base = after.astimezone(timezone.utc)
    itr = croniter(cron_expression, base)
    nxt = itr.get_next(datetime)
    if nxt.tzinfo is None:
        nxt = nxt.replace(tzinfo=timezone.utc)
    return nxt.astimezone(timezone.utc)


class Handlers:
    def __init__(self, store: DueStore):
        self.store = store

    def dispatch(self, message: dict[str, Any]) -> None:
        task = message.get("task") or ""
        args = message.get("args") or []
        if task == "worker.schedule_task":
            self.schedule_task(*args)
        elif task == "worker.delete_task":
            self.delete_task(*args)
        elif task in (
            "worker.create_digest_settings",
            "worker.update_digest_settings",
            "worker.delete_digest_settings",
        ):
            log.info("digest task ignored in sample worker: %s args=%s", task, args)
        else:
            log.warning("unknown task=%s args=%s", task, args)

    def schedule_task(
        self,
        messenger_name: str,
        chat_id: str,
        task_id: Any,
        task_title: str,
        task_description: str = "",
        scheduled_time_str: Any = None,
        cron_expression: Any = None,
        requires_confirmation: Any = False,
    ) -> None:
        jid = job_id(str(messenger_name), task_id)
        cron = cron_expression if cron_expression not in (None, "", "null") else None
        start = parse_datetime(scheduled_time_str)

        if not start and not cron:
            raise ValueError("schedule_task needs start_date and/or cron_expression")

        # One-shot or first occurrence: use start_date when present.
        if start is None and cron:
            start = _next_cron(str(cron), datetime.now(timezone.utc))

        assert start is not None
        now = datetime.now(timezone.utc)
        # If already overdue and recurring, jump to next cron tick.
        if cron and start <= now:
            start = _next_cron(str(cron), now)

        payload = {
            "messenger_name": str(messenger_name),
            "chat_id": str(chat_id),
            "task_id": task_id,
            "task_title": task_title,
            "task_description": task_description or "",
            "cron_expression": cron,
            "requires_confirmation": _as_bool(requires_confirmation),
        }
        self.store.upsert(jid, start, payload)

    def delete_task(self, task_id: Any, messenger_name: str = "telegram") -> None:
        self.store.remove(job_id(str(messenger_name), task_id))

    def fire_due(self) -> int:
        """Claim and deliver due jobs; re-arm cron jobs for the next tick."""
        fired = 0
        for jid, payload in self.store.due():
            if not self.store.claim(jid):
                continue
            try:
                send_reminder(payload)
                fired += 1
            except Exception:
                # Put back ~30s later so a down webhook can recover.
                retry_at = datetime.fromtimestamp(
                    datetime.now(timezone.utc).timestamp() + 30,
                    tz=timezone.utc,
                )
                self.store.upsert(jid, retry_at, payload)
                continue

            cron = payload.get("cron_expression")
            if cron:
                nxt = _next_cron(str(cron), datetime.now(timezone.utc))
                self.store.upsert(jid, nxt, payload)
            else:
                self.store.remove(jid)
        return fired
