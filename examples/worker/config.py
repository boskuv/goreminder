"""Env config for the sample worker."""

from __future__ import annotations

import os


def _env(name: str, default: str) -> str:
    return os.environ.get(name, default)


BROKER_URL = _env("BROKER_URL", "amqp://guest:guest@localhost:5672/")
QUEUE_NAME = _env("QUEUE_NAME", "celery")
EXCHANGE = _env("EXCHANGE", "celery")  # Go publisher uses direct exchange = queue name
REDIS_URL = _env("REDIS_URL", "redis://localhost:6379/0")
WEBHOOK_URL = _env("WEBHOOK_URL", "http://localhost:8001/send_message")

# Redis keys
ZSET_KEY = _env("GOREMINDER_ZSET_KEY", "goreminder:due")
JOB_HASH_PREFIX = _env("GOREMINDER_JOB_PREFIX", "goreminder:job:")
ANTI_DUPE_TTL = int(_env("ANTI_DUPE_TTL", "30"))
POLL_INTERVAL_SEC = float(_env("POLL_INTERVAL_SEC", "1.0"))
WEBHOOK_TIMEOUT_SEC = float(_env("WEBHOOK_TIMEOUT_SEC", "10"))
