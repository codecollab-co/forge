"""forge-agent entry point.

Surface:
- GET /healthz                 — liveness
- GET /runs/:id/stream         — SSE live trace (slice 9)
- POST /verify-token           — verifies a forge-platform JWT (legacy)
- Background consumer          — pulls run.requested events, drives Runs
"""

from __future__ import annotations

import asyncio
import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

from app.auth import AuthError, verify
from app.eventbus import Consumer
from app.platform_client import PlatformClient
from app.run_pubsub import (
    RunEvent,
    RunPubSub,
    fetch_history,
    is_terminal,
    make_pubsub,
)
from app.runner import RunRequest, Runner
from app.sandbox import from_env as sandbox_from_env

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("forge-agent")

WEBSITE_DOMAIN = os.environ.get("WEBSITE_DOMAIN", "http://localhost:3000")


@asynccontextmanager
async def lifespan(app: FastAPI):
    pubsub = make_pubsub()
    runner = Runner(
        sandboxes=sandbox_from_env(),
        platform=PlatformClient(),
        pubsub=pubsub,
    )
    consumer = Consumer()
    app.state.pubsub = pubsub

    async def handle(event_type: str, payload: dict) -> None:
        if event_type != "run.requested":
            return
        if int(payload.get("v", 0)) != 1:
            logger.warning("ignoring run.requested with unknown version v=%s", payload.get("v"))
            return
        try:
            req = RunRequest(
                run_id=payload["run_id"],
                repo_id=payload["repo_id"],
                repo_owner=payload["repo_owner"],
                repo_name=payload["repo_name"],
                issue_id=payload["issue_id"],
                issue_number=int(payload["issue_number"]),
                issue_title=payload.get("issue_title", ""),
                issue_body=payload.get("issue_body", ""),
                requested_by=payload["requested_by"],
            )
        except Exception:
            logger.exception("malformed run.requested payload: %r", payload)
            return
        asyncio.create_task(runner.run(req))

    task = asyncio.create_task(consumer.run(handle))
    try:
        yield
    finally:
        consumer.stop()
        await asyncio.wait_for(task, timeout=5)


app = FastAPI(lifespan=lifespan)
app.add_middleware(
    CORSMiddleware,
    allow_origins=[WEBSITE_DOMAIN],
    allow_credentials=True,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["*"],
)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok", "service": "forge-agent"}


class TokenIn(BaseModel):
    token: str


@app.post("/verify-token")
def verify_token(body: TokenIn) -> dict:
    try:
        claims = verify(body.token)
    except AuthError as exc:
        raise HTTPException(status_code=401, detail=str(exc)) from exc
    return {"valid": True, "claims": claims}


@app.get("/runs/{run_id}/stream")
async def stream_run(run_id: str, request: Request) -> StreamingResponse:
    pubsub: RunPubSub = request.app.state.pubsub
    last_event_id_header = request.headers.get("Last-Event-ID", "0")
    try:
        since_id = int(last_event_id_header)
    except ValueError:
        since_id = 0

    async def gen():
        # 1. Replay historical events (catch up from since_id).
        try:
            history = await asyncio.to_thread(
                fetch_history, os.environ["DATABASE_URL"], run_id, since_id
            )
        except Exception:
            logger.exception("history fetch failed for run=%s", run_id)
            history = []
        for evt in history:
            yield evt.to_sse()
            if is_terminal(evt.type):
                return

        # 2. Subscribe and stream live events.
        queue = await pubsub.subscribe(run_id)
        try:
            while True:
                if await request.is_disconnected():
                    return
                try:
                    item = await asyncio.wait_for(queue.get(), timeout=15.0)
                except asyncio.TimeoutError:
                    yield ": ping\n\n"
                    continue
                if item is None:
                    return
                yield item.to_sse()
                if is_terminal(item.type):
                    return
        finally:
            await pubsub.unsubscribe(run_id, queue)

    return StreamingResponse(
        gen(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",  # for nginx
        },
    )


def _port() -> int:
    return int(os.environ.get("PORT", "8081"))


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=_port(), log_level="info")
