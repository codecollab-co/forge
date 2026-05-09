"""In-process pub/sub for live Run events.

Single-instance assumption today: the same orchestrator process publishes
(via the Runner) and subscribes (via the SSE endpoint). When we scale the
orchestrator horizontally, swap this for Postgres LISTEN/NOTIFY or Redis
Streams — the surface is intentionally small to keep the swap mechanical.
"""

from __future__ import annotations

import asyncio
import logging
from collections import defaultdict
from dataclasses import dataclass
from typing import Any

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class RunEvent:
    type: str
    payload: dict
    db_id: int | None = None  # populated once persisted by the platform

    def to_sse(self) -> str:
        import json

        body = json.dumps({"type": self.type, "payload": self.payload})
        if self.db_id is not None:
            return f"id: {self.db_id}\nevent: {self.type}\ndata: {body}\n\n"
        return f"event: {self.type}\ndata: {body}\n\n"


class RunPubSub:
    """asyncio.Queue per run_id. Subscribers receive only future events; use
    the historical replay path (DB) to fill in the gap on reconnect."""

    def __init__(self) -> None:
        self._subs: dict[str, set[asyncio.Queue[RunEvent | None]]] = defaultdict(set)
        self._lock = asyncio.Lock()

    async def publish(self, run_id: str, event: RunEvent) -> None:
        async with self._lock:
            queues = list(self._subs.get(run_id, ()))
        for q in queues:
            try:
                q.put_nowait(event)
            except asyncio.QueueFull:
                logger.warning("dropping event for slow subscriber on run=%s", run_id)

    async def close(self, run_id: str) -> None:
        """Signal end-of-stream by sending a sentinel (None)."""
        async with self._lock:
            queues = list(self._subs.get(run_id, ()))
        for q in queues:
            try:
                q.put_nowait(None)
            except asyncio.QueueFull:
                pass

    async def subscribe(self, run_id: str) -> asyncio.Queue[RunEvent | None]:
        q: asyncio.Queue[RunEvent | None] = asyncio.Queue(maxsize=128)
        async with self._lock:
            self._subs[run_id].add(q)
        return q

    async def unsubscribe(self, run_id: str, q: asyncio.Queue[RunEvent | None]) -> None:
        async with self._lock:
            if run_id in self._subs:
                self._subs[run_id].discard(q)
                if not self._subs[run_id]:
                    del self._subs[run_id]


# Process-global instance; injected via FastAPI app.state in main.py.
def make_pubsub() -> RunPubSub:
    return RunPubSub()


async def fetch_history(dsn: str, run_id: str, since_id: int = 0) -> list[RunEvent]:
    """Reads agent.run_events with id > since_id. Used on SSE reconnect."""
    import psycopg

    out: list[RunEvent] = []
    with psycopg.connect(dsn) as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT id, type, payload
                  FROM agent.run_events
                 WHERE run_id = %s AND id > %s
                 ORDER BY id ASC
                """,
                (run_id, since_id),
            )
            for db_id, evt_type, payload in cur.fetchall():
                out.append(
                    RunEvent(
                        type=evt_type,
                        payload=payload if isinstance(payload, dict) else {},
                        db_id=int(db_id),
                    )
                )
    return out


def is_terminal(event_type: str) -> bool:
    """An event marking the Run as done (succeeded / failed / cancelled).
    Subscribers should close after seeing one of these."""
    return event_type.startswith("run.terminal")
