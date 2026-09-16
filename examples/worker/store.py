"""Redis due-time store: ZSET of fire times + HASH of job payloads."""

from __future__ import annotations

import json
import logging
from datetime import datetime, timezone
from typing import Any

import redis

from config import JOB_HASH_PREFIX, REDIS_URL, ZSET_KEY

log = logging.getLogger("goreminder-worker.store")


def job_id(messenger_name: str, task_id: int | str) -> str:
    return f"{messenger_name}_{task_id}"


class DueStore:
    """Schedule reminders as score=unix_ts members in a Redis ZSET."""

    def __init__(self, url: str = REDIS_URL):
        self.r = redis.Redis.from_url(url, decode_responses=True)

    def _hash_key(self, jid: str) -> str:
        return f"{JOB_HASH_PREFIX}{jid}"

    def upsert(
        self,
        jid: str,
        fire_at: datetime,
        payload: dict[str, Any],
    ) -> None:
        if fire_at.tzinfo is None:
            fire_at = fire_at.replace(tzinfo=timezone.utc)
        score = fire_at.timestamp()
        pipe = self.r.pipeline()
        pipe.hset(self._hash_key(jid), mapping={"payload": json.dumps(payload)})
        pipe.zadd(ZSET_KEY, {jid: score})
        pipe.execute()
        log.info("scheduled job_id=%s fire_at=%s", jid, fire_at.isoformat())

    def remove(self, jid: str) -> bool:
        pipe = self.r.pipeline()
        pipe.zrem(ZSET_KEY, jid)
        pipe.delete(self._hash_key(jid))
        removed, _ = pipe.execute()
        log.info("removed job_id=%s existed=%s", jid, bool(removed))
        return bool(removed)

    def due(self, now: datetime | None = None, limit: int = 64) -> list[tuple[str, dict[str, Any]]]:
        if now is None:
            now = datetime.now(timezone.utc)
        max_score = now.timestamp()
        ids = self.r.zrangebyscore(ZSET_KEY, "-inf", max_score, start=0, num=limit)
        out: list[tuple[str, dict[str, Any]]] = []
        for jid in ids:
            raw = self.r.hget(self._hash_key(jid), "payload")
            if not raw:
                self.r.zrem(ZSET_KEY, jid)
                continue
            out.append((jid, json.loads(raw)))
        return out

    def claim(self, jid: str) -> bool:
        """Atomically drop from ZSET so only one poller fires the job."""
        return bool(self.r.zrem(ZSET_KEY, jid))
