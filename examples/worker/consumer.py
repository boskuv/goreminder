"""AMQP consumer for GoReminder Celery-style JSON on the celery queue."""

from __future__ import annotations

import json
import logging
import time
from typing import Callable

import pika

from config import BROKER_URL, EXCHANGE, QUEUE_NAME

log = logging.getLogger("goreminder-worker.consumer")


class QueueConsumer:
    def __init__(self, on_message: Callable[[dict], None]):
        self.on_message = on_message
        self._stopping = False

    def stop(self) -> None:
        self._stopping = True

    def run_forever(self) -> None:
        while not self._stopping:
            try:
                self._consume_once()
            except Exception:
                log.exception("consumer disconnected; retry in 3s")
                time.sleep(3)

    def _consume_once(self) -> None:
        params = pika.URLParameters(BROKER_URL)
        connection = pika.BlockingConnection(params)
        channel = connection.channel()
        channel.exchange_declare(exchange=EXCHANGE, exchange_type="direct", durable=True)
        channel.queue_declare(queue=QUEUE_NAME, durable=True)
        channel.queue_bind(queue=QUEUE_NAME, exchange=EXCHANGE, routing_key=QUEUE_NAME)
        channel.basic_qos(prefetch_count=1)

        def _callback(ch, method, _properties, body: bytes) -> None:
            try:
                payload = json.loads(body.decode("utf-8"))
                log.info("recv task=%s", payload.get("task"))
                self.on_message(payload)
                ch.basic_ack(delivery_tag=method.delivery_tag)
            except Exception:
                log.exception("handler failed; nack requeue=false body=%s", body[:500])
                ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

        channel.basic_consume(queue=QUEUE_NAME, on_message_callback=_callback)
        log.info("consuming queue=%s exchange=%s broker=%s", QUEUE_NAME, EXCHANGE, BROKER_URL)
        try:
            while not self._stopping:
                connection.process_data_events(time_limit=1)
        finally:
            try:
                connection.close()
            except Exception:
                pass
