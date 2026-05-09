"""forge-agent entry point.

Slice-1 surface:
- GET /healthz             — liveness
- POST /verify-token       — proves the JWT seam (verifies a token issued by forge-platform)
- Background consumer      — pulls events from platform.events and logs them
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

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("forge-agent")


async def _handle(event_type: str, payload: dict) -> None:
    logger.info("event received type=%s payload=%s", event_type, payload)


@asynccontextmanager
async def lifespan(_: FastAPI):
    consumer = Consumer()
    task = asyncio.create_task(consumer.run(_handle))
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
