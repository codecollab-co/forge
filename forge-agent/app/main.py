"""forge-agent entry point.

Slice-7 surface:
- GET /healthz             — liveness
- POST /verify-token       — verifies a forge-platform JWT (legacy from slice 1)
- Background consumer      — pulls run.requested events and runs them
"""

from __future__ import annotations

import asyncio
import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from app.auth import AuthError, verify
from app.eventbus import Consumer
from app.platform_client import PlatformClient
from app.runner import RunRequest, Runner
from app.sandbox import from_env as sandbox_from_env

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("forge-agent")


def _build_runner() -> Runner:
    return Runner(sandboxes=sandbox_from_env(), platform=PlatformClient())


@asynccontextmanager
async def lifespan(_: FastAPI):
    runner = _build_runner()
    consumer = Consumer()

    async def handle(event_type: str, payload: dict) -> None:
        if event_type != "run.requested":
            logger.debug("ignoring event type=%s", event_type)
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
        # Run-per-event in the background so the consumer keeps draining.
        asyncio.create_task(runner.run(req))

    task = asyncio.create_task(consumer.run(handle))
    try:
        yield
    finally:
        consumer.stop()
        await asyncio.wait_for(task, timeout=5)


app = FastAPI(lifespan=lifespan)


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


def _port() -> int:
    return int(os.environ.get("PORT", "8081"))


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=_port(), log_level="info")
