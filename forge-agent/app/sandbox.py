"""SandboxProvider — the deep module from ADR-0005.

Acquire returns an opaque Sandbox handle; Release ends it. The interface is
deliberately narrow so swapping E2B for an in-house Firecracker fleet is a
single-module change.

Backends:
  - InMemoryFake: used by tests and as the local-dev default when E2B_API_KEY
    is unset. No real isolation; returns synthetic IDs.
  - E2BProvider: real E2B sandboxes via the e2b SDK. Workspace I/O is sync
    and called from runner.py via asyncio.to_thread so the event loop and
    heartbeat keep ticking.
"""

from __future__ import annotations

import os
import uuid
from dataclasses import dataclass, field
from typing import Any, Protocol


@dataclass
class Sandbox:
    id: str
    backend: str   # "in-memory" | "e2b" | ...
    handle: Any = None  # opaque; the workspace knows what to do with it


class SandboxProvider(Protocol):
    async def acquire(self, *, run_id: str) -> Sandbox: ...
    async def release(self, sandbox: Sandbox) -> None: ...


class InMemoryFake:
    """Deterministic, no-op sandbox. Used by unit tests and local dev when
    E2B_API_KEY is empty."""

    def __init__(self) -> None:
        self.acquired: list[str] = []
        self.released: list[str] = []

    async def acquire(self, *, run_id: str) -> Sandbox:
        sid = f"sbx-fake-{uuid.uuid4().hex[:8]}"
        self.acquired.append(sid)
        return Sandbox(id=sid, backend="in-memory")

    async def release(self, sandbox: Sandbox) -> None:
        self.released.append(sandbox.id)


class E2BProvider:
    """Real E2B sandboxes. Lazily imports the SDK so the agent boots even
    when e2b isn't installed (relevant for tests)."""

    def __init__(self, api_key: str, timeout_s: int = 30 * 60) -> None:
        self._api_key = api_key
        self._timeout = timeout_s

    async def acquire(self, *, run_id: str) -> Sandbox:
        import asyncio
        from e2b import Sandbox as E2BSandbox

        def _create() -> Any:
            return E2BSandbox(api_key=self._api_key, timeout=self._timeout)

        sb = await asyncio.to_thread(_create)
        return Sandbox(id=sb.sandbox_id, backend="e2b", handle=sb)

    async def release(self, sandbox: Sandbox) -> None:
        import asyncio
        if sandbox.handle is None:
            return
        await asyncio.to_thread(sandbox.handle.kill)


def from_env() -> SandboxProvider:
    """Selects the backend from env. Empty E2B_API_KEY -> InMemoryFake."""
    if os.environ.get("E2B_API_KEY"):
        return E2BProvider(api_key=os.environ["E2B_API_KEY"])
    return InMemoryFake()
