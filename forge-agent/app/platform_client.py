"""HTTP client for forge-platform's /internal/* endpoints.

Auth: short-lived RS256 JWT signed with the same private key forge-platform
uses (we sign here rather than fetching a token, for simplicity at MVP).
"""

from __future__ import annotations

import datetime as dt
import logging
import os
from typing import Any

import httpx
import jwt

logger = logging.getLogger(__name__)


def _private_key() -> str:
    raw = os.environ.get("JWT_PRIVATE_KEY", "").strip()
    if not raw:
        raise RuntimeError("JWT_PRIVATE_KEY is empty")
    return raw.replace("\\n", "\n")


def _service_token() -> str:
    now = dt.datetime.now(dt.timezone.utc)
    payload = {
        "iss": "forge-platform",
        "sub": "forge-agent",
        "iat": int(now.timestamp()),
        "exp": int((now + dt.timedelta(minutes=5)).timestamp()),
    }
    return jwt.encode(payload, _private_key(), algorithm="RS256")


class PlatformClient:
    def __init__(self, base_url: str | None = None, *, timeout: float = 15.0) -> None:
        self._base_url = (
            base_url
            or os.environ.get("PLATFORM_API_URL", "http://forge-platform:8080")
        ).rstrip("/")
        self._timeout = timeout

    def _headers(self) -> dict[str, str]:
        return {
            "Authorization": f"Bearer {_service_token()}",
            "Content-Type": "application/json",
        }

    async def update_run_state(self, run_id: str, **fields: Any) -> None:
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            r = await c.post(
                f"{self._base_url}/internal/runs/{run_id}/state",
                headers=self._headers(),
                json=fields,
            )
            r.raise_for_status()

    async def heartbeat(self, run_id: str) -> bool:
        """Returns whether cancel has been requested."""
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            r = await c.post(
                f"{self._base_url}/internal/runs/{run_id}/heartbeat",
                headers=self._headers(),
            )
            r.raise_for_status()
            return bool(r.json().get("cancel_requested", False))

    async def append_event(self, run_id: str, event_type: str, payload: dict | None = None) -> None:
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            r = await c.post(
                f"{self._base_url}/internal/runs/{run_id}/events",
                headers=self._headers(),
                json={"type": event_type, "payload": payload or {}},
            )
            r.raise_for_status()

    async def commit(
        self,
        repo_id: str,
        *,
        branch: str,
        base_branch: str,
        files: list[dict],
        message: str,
        author: dict,
    ) -> str:
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            r = await c.post(
                f"{self._base_url}/internal/repos/{repo_id}/commits",
                headers=self._headers(),
                json={
                    "branch": branch,
                    "base_branch": base_branch,
                    "files": files,
                    "message": message,
                    "author": author,
                },
            )
            r.raise_for_status()
            return r.json()["commit_oid"]

    async def open_pr(
        self,
        repo_id: str,
        *,
        title: str,
        body: str,
        head_branch: str,
        base_branch: str,
        author_id: str,
        run_id: str,
    ) -> dict:
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            r = await c.post(
                f"{self._base_url}/internal/repos/{repo_id}/pulls",
                headers=self._headers(),
                json={
                    "title": title,
                    "body": body,
                    "head_branch": head_branch,
                    "base_branch": base_branch,
                    "author_id": author_id,
                    "run_id": run_id,
                },
            )
            r.raise_for_status()
            return r.json()
