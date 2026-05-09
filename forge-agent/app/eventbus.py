"""Postgres-as-queue consumer (per ADR-0009).

Polls platform.events with SELECT ... FOR UPDATE SKIP LOCKED. The interface is
deliberately narrow so the backend can be swapped (Redis Streams / SQS) later
without touching the orchestrator.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
from collections.abc import Awaitable, Callable

import psycopg

logger = logging.getLogger(__name__)

Handler = Callable[[str, dict], Awaitable[None]]


class Consumer:
    def __init__(self, dsn: str | None = None, poll_interval: float = 1.0):
        self._dsn = dsn or os.environ["DATABASE_URL"]
        self._poll_interval = poll_interval
        self._stop = asyncio.Event()

    def stop(self) -> None:
        self._stop.set()

    async def run(self, handler: Handler) -> None:
        while not self._stop.is_set():
            try:
                claimed = await self._claim_one(handler)
                if not claimed:
                    await asyncio.sleep(self._poll_interval)
            except Exception:
                logger.exception("eventbus consumer error")
                await asyncio.sleep(self._poll_interval)

    async def _claim_one(self, handler: Handler) -> bool:
        with psycopg.connect(self._dsn, autocommit=False) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    SELECT id, type, payload
                      FROM platform.events
                     WHERE status = 'pending'
                     ORDER BY id
                     LIMIT 1
                     FOR UPDATE SKIP LOCKED
                    """
                )
                row = cur.fetchone()
                if not row:
                    conn.rollback()
                    return False
                event_id, event_type, payload = row
                payload_dict = payload if isinstance(payload, dict) else json.loads(payload)
                try:
                    await handler(event_type, payload_dict)
                except Exception:
                    logger.exception("handler failed for event %s", event_id)
                    cur.execute(
                        "UPDATE platform.events SET status = 'failed' WHERE id = %s",
                        (event_id,),
                    )
                    conn.commit()
                    return True
                cur.execute(
                    "UPDATE platform.events SET status = 'consumed', consumed_at = NOW() WHERE id = %s",
                    (event_id,),
                )
                conn.commit()
                logger.info("consumed event id=%s type=%s", event_id, event_type)
                return True
