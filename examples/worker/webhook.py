"""POST reminder payloads to the messenger webhook (e.g. telegram-bot /send_message)."""

from __future__ import annotations

import logging
from typing import Any

import redis
import requests

from config import ANTI_DUPE_TTL, REDIS_URL, WEBHOOK_TIMEOUT_SEC, WEBHOOK_URL

log = logging.getLogger("goreminder-worker.webhook")


def build_payload(job: dict[str, Any]) -> dict[str, Any]:
    title = str(job.get("task_title") or "").replace("🔔", "").strip()
    task_id = job.get("task_id")
    is_pre = bool(job.get("is_pre_remind"))
    if is_pre:
        text = f"⏳ Скоро: {title}" if title else "⏳ Скоро напоминание"
    else:
        text = f"🔔 {title}" if title else "🔔 reminder"
    payload: dict[str, Any] = {
        "chat_id": str(job.get("chat_id") or ""),
        "text": text,
        "task_id": task_id,
    }
    # Pre-remind never includes done/later buttons.
    if job.get("requires_confirmation") and not is_pre:
        payload["reply_markup"] = {
            "inline_keyboard": [
                [
                    {"text": "✅ Выполнено", "callback_data": f"done:{task_id}"},
                    {"text": "⏰ Отложить", "callback_data": f"later:{task_id}"},
                ]
            ]
        }
    return payload


def send_reminder(job: dict[str, Any], *, webhook_url: str = WEBHOOK_URL) -> bool:
    chat_id = str(job.get("chat_id") or "")
    task_id = job.get("task_id")
    kind = "pre" if job.get("is_pre_remind") else "main"
    r = redis.Redis.from_url(REDIS_URL, decode_responses=True)
    cache_key = f"anti_dupe:{chat_id}:{task_id or job.get('task_title')}:{kind}"
    if not r.set(cache_key, "1", ex=ANTI_DUPE_TTL, nx=True):
        log.warning("skip duplicate send key=%s", cache_key)
        return True

    payload = build_payload(job)
    try:
        resp = requests.post(
            webhook_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=WEBHOOK_TIMEOUT_SEC,
        )
        resp.raise_for_status()
        log.info(
            "webhook ok chat_id=%s task_id=%s kind=%s status=%s",
            chat_id,
            task_id,
            kind,
            resp.status_code,
        )
        return True
    except Exception:
        r.delete(cache_key)
        log.exception(
            "webhook failed chat_id=%s task_id=%s kind=%s url=%s",
            chat_id,
            task_id,
            kind,
            webhook_url,
        )
        raise
