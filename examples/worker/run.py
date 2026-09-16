"""
Sample GoReminder worker (not Celery).

Flow:
  RabbitMQ (celery queue, Celery-style JSON)
    → handlers.schedule_task / delete_task
    → Redis ZSET (due) + HASH (payload)
    → poller fires POST WEBHOOK_URL (/send_message)

Env: see config.py / README.md
"""

from __future__ import annotations

import logging
import signal
import threading

from config import POLL_INTERVAL_SEC, REDIS_URL, WEBHOOK_URL
from consumer import QueueConsumer
from handlers import Handlers
from store import DueStore

logging.basicConfig(
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
    level=logging.INFO,
)
log = logging.getLogger("goreminder-worker")


def main() -> int:
    store = DueStore(REDIS_URL)
    handlers = Handlers(store)
    consumer = QueueConsumer(handlers.dispatch)
    stop = threading.Event()

    def _poll() -> None:
        while not stop.is_set():
            try:
                n = handlers.fire_due()
                if n:
                    log.info("fired %s due job(s)", n)
            except Exception:
                log.exception("poller error")
            stop.wait(POLL_INTERVAL_SEC)

    def _shutdown(*_args) -> None:
        log.info("shutting down")
        stop.set()
        consumer.stop()

    signal.signal(signal.SIGINT, _shutdown)
    signal.signal(signal.SIGTERM, _shutdown)

    poller = threading.Thread(target=_poll, name="due-poller", daemon=True)
    poller.start()
    log.info("worker up webhook=%s redis=%s", WEBHOOK_URL, REDIS_URL)

    try:
        consumer.run_forever()
    finally:
        stop.set()
        poller.join(timeout=2)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
