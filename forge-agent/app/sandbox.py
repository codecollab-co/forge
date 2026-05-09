"""SandboxProvider — the deep module from ADR-0005.

Acquire returns an opaque Sandbox handle; Release ends it. The interface is
deliberately narrow so swapping E2B for an in-house Firecracker fleet is a
single-module change.

Backends:
  - InMemoryFake: used by tests and as the local-dev default when E2B_API_KEY
    is unset. No real isolation; returns synthetic IDs.
  - E2BProvider: real E2B sandboxes. **Wired but not yet implemented** —
    the slice-7 stub doesn't actually execute anything inside the sandbox,
    so we keep the integration as a follow-up. The protocol is here so
    swapping implementations is a one-line change in the factory below.
"""

from __future__ import annotations

import os
import uuid
from dataclasses import dataclass
from typing import Protocol


@dataclass(frozen=True)
class Sandbox:
    id: str
    backend: str  # "in-memory" | "e2b" | ...


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
    """Placeholder. Real implementation lands as a small follow-up: the slice-7
    stub does its file write through the platform-api internal commit endpoint,
    so it doesn't yet need to execute inside an E2B sandbox. When slice 8 lands
    the real Agent loop, this becomes a thin wrapper over `e2b_code_interpreter`."""

    def __init__(self, api_key: str) -> None:
        self._api_key = api_key

    async def acquire(self, *, run_id: str) -> Sandbox:  # pragma: no cover - placeholder
        raise NotImplementedError("E2BProvider.acquire — lands with slice 8")

    async def release(self, sandbox: Sandbox) -> None:  # pragma: no cover
        raise NotImplementedError("E2BProvider.release — lands with slice 8")


def from_env() -> SandboxProvider:
    """Selects the backend from env. Empty E2B_API_KEY -> InMemoryFake."""
    if os.environ.get("E2B_API_KEY"):
        return E2BProvider(api_key=os.environ["E2B_API_KEY"])
    return InMemoryFake()
