"""Dispatch Celery-style {task, args} messages into the Redis due-store."""

from __future__ import annotations

import logging
from datetime import datetime, timedelta, timezone
from typing import Any

from croniter import croniter
from dateutil.rrule import rrulestr

from datetime_util import parse_datetime
from store import DueStore, job_id, pre_job_id
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


def _as_positive_int(v: Any) -> int | None:
    if v is None or v == "" or v == "null":
        return None
    try:
        n = int(v)
    except (TypeError, ValueError):
        return None
    return n if n > 0 else None


def _optional_str(v: Any) -> str | None:
    if v in (None, "", "null"):
        return None
    return str(v)


def _next_cron(cron_expression: str, after: datetime) -> datetime:
    """Next fire time. GoReminder uses 5-field cron; croniter accepts that."""
    base = after.astimezone(timezone.utc)
    itr = croniter(cron_expression, base)
    nxt = itr.get_next(datetime)
    if nxt.tzinfo is None:
        nxt = nxt.replace(tzinfo=timezone.utc)
    return nxt.astimezone(timezone.utc)


def _next_rrule(rrule_expression: str, after: datetime, dtstart: datetime | None = None) -> datetime:
    """Next occurrence strictly after `after`, using start_date (or after) as DTSTART."""
    base = after.astimezone(timezone.utc)
    start = (dtstart or base).astimezone(timezone.utc)
    rule = rrulestr(rrule_expression, dtstart=start)
    nxt = rule.after(base, inc=False)
    if nxt is None:
        raise ValueError(f"no rrule occurrence after {base.isoformat()}")
    if nxt.tzinfo is None:
        nxt = nxt.replace(tzinfo=timezone.utc)
    return nxt.astimezone(timezone.utc)


def _next_recurrence(
    cron: str | None,
    rrule: str | None,
    after: datetime,
    dtstart: datetime | None = None,
) -> datetime | None:
    """Prefer cron when both are set (domain treats them as mutually exclusive)."""
    if cron:
        return _next_cron(cron, after)
    if rrule:
        return _next_rrule(rrule, after, dtstart=dtstart)
    return None


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
        pre_remind_before_seconds: Any = None,
        rrule: Any = None,
    ) -> None:
        jid = job_id(str(messenger_name), task_id)
        pjid = pre_job_id(str(messenger_name), task_id)
        cron = _optional_str(cron_expression)
        rule = _optional_str(rrule)
        start = parse_datetime(scheduled_time_str)
        pre_secs = _as_positive_int(pre_remind_before_seconds)

        if not start and not cron and not rule:
            raise ValueError("schedule_task needs start_date and/or cron_expression or rrule")

        # One-shot or first occurrence: use start_date when present.
        if start is None:
            start = _next_recurrence(cron, rule, datetime.now(timezone.utc))

        assert start is not None
        now = datetime.now(timezone.utc)
        # If already overdue and recurring, jump to next tick.
        if (cron or rule) and start <= now:
            start = _next_recurrence(cron, rule, now, dtstart=start)

        payload = {
            "messenger_name": str(messenger_name),
            "chat_id": str(chat_id),
            "task_id": task_id,
            "task_title": task_title,
            "task_description": task_description or "",
            "cron_expression": cron,
            "rrule": rule,
            "dtstart": start.isoformat(),
            "requires_confirmation": _as_bool(requires_confirmation),
            "pre_remind_before_seconds": pre_secs,
            "is_pre_remind": False,
        }
        self.store.upsert(jid, start, payload)
        self._sync_pre_remind(pjid, start, payload, pre_secs, now)

    def _sync_pre_remind(
        self,
        pjid: str,
        main_fire_at: datetime,
        main_payload: dict[str, Any],
        pre_secs: int | None,
        now: datetime,
    ) -> None:
        """Upsert or clear the preliminary reminder job relative to main_fire_at."""
        if not pre_secs:
            self.store.remove(pjid)
            return
        pre_at = main_fire_at - timedelta(seconds=pre_secs)
        if pre_at <= now:
            log.info(
                "skip pre-remind job_id=%s (fire_at=%s already past)",
                pjid,
                pre_at.isoformat(),
            )
            self.store.remove(pjid)
            return
        pre_payload = {
            **main_payload,
            "cron_expression": None,  # pre-remind is one-shot per occurrence
            "rrule": None,
            "requires_confirmation": False,
            "is_pre_remind": True,
        }
        self.store.upsert(pjid, pre_at, pre_payload)

    def delete_task(self, task_id: Any, messenger_name: str = "telegram") -> None:
        self.store.remove(job_id(str(messenger_name), task_id))
        self.store.remove(pre_job_id(str(messenger_name), task_id))

    def fire_due(self) -> int:
        """Claim and deliver due jobs; re-arm cron/rrule jobs for the next tick."""
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

            if payload.get("is_pre_remind"):
                self.store.remove(jid)
                continue

            cron = _optional_str(payload.get("cron_expression"))
            rule = _optional_str(payload.get("rrule"))
            pre_secs = _as_positive_int(payload.get("pre_remind_before_seconds"))
            messenger = str(payload.get("messenger_name") or "telegram")
            task_id = payload.get("task_id")
            dtstart = parse_datetime(payload.get("dtstart"))
            nxt = _next_recurrence(cron, rule, datetime.now(timezone.utc), dtstart=dtstart)
            if nxt is not None:
                self.store.upsert(jid, nxt, payload)
                self._sync_pre_remind(
                    pre_job_id(messenger, task_id),
                    nxt,
                    payload,
                    pre_secs,
                    datetime.now(timezone.utc),
                )
            else:
                self.store.remove(jid)
                # One-shot main fired: drop any leftover pre (should already be gone).
                self.store.remove(pre_job_id(messenger, task_id))
        return fired
