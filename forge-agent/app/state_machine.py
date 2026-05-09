"""RunStateMachine — the deep module from ADR-0007.

Pure logic. No I/O, no globals, no side effects. Inputs are events; outputs
are the next state plus a list of side-effect intents that the caller (the
Runner) translates into actual API calls.

Tested in isolation in state_machine_test.py with no model, no sandbox, no
HTTP, no DB.
"""

from __future__ import annotations

from dataclasses import dataclass, field, replace
from enum import Enum
from typing import Literal, Sequence


class State(str, Enum):
    QUEUED = "queued"
    RUNNING = "running"
    SUCCEEDED = "succeeded"
    FAILED = "failed"
    CANCELLED = "cancelled"


# Events
@dataclass(frozen=True)
class StartRequested:
    pass


@dataclass(frozen=True)
class SandboxAcquired:
    sandbox_id: str


@dataclass(frozen=True)
class SandboxAcquireFailed:
    reason: str


@dataclass(frozen=True)
class StubEditPrepared:
    pass


@dataclass(frozen=True)
class PROpened:
    pr_id: str
    pr_number: int


@dataclass(frozen=True)
class CancelRequested:
    pass


@dataclass(frozen=True)
class CrashedOrCancelled:
    """Generic terminal-failure signal (e.g. unexpected exception)."""
    category: str
    message: str


Event = (
    StartRequested
    | SandboxAcquired
    | SandboxAcquireFailed
    | StubEditPrepared
    | PROpened
    | CancelRequested
    | CrashedOrCancelled
)


# Side-effect intents
@dataclass(frozen=True)
class AcquireSandbox:
    pass


@dataclass(frozen=True)
class PrepareStubEdit:
    sandbox_id: str


@dataclass(frozen=True)
class CommitAndOpenPR:
    sandbox_id: str


@dataclass(frozen=True)
class ReleaseSandbox:
    sandbox_id: str | None
    reason: Literal["succeeded", "failed", "cancelled"]


Intent = AcquireSandbox | PrepareStubEdit | CommitAndOpenPR | ReleaseSandbox


@dataclass(frozen=True)
class Context:
    """The mutable bits of a Run that the state machine tracks alongside the
    State enum. Held outside the State so the state symbol stays small."""

    state: State = State.QUEUED
    sandbox_id: str | None = None
    pr_id: str | None = None
    pr_number: int | None = None
    error_category: str | None = None
    error_message: str | None = None


@dataclass(frozen=True)
class Step:
    context: Context
    intents: Sequence[Intent] = field(default_factory=tuple)


def step(ctx: Context, event: Event) -> Step:
    """Pure transition. Always returns a new Context (frozen dataclass)."""

    # CancelRequested is accepted from any non-terminal state; it asks to
    # gracefully stop. The terminal transition happens once the side-effect
    # intent (ReleaseSandbox) is acknowledged via CrashedOrCancelled('cancel'...).
    if isinstance(event, CancelRequested):
        if ctx.state in (State.SUCCEEDED, State.FAILED, State.CANCELLED):
            return Step(ctx)
        return Step(
            replace(ctx, state=State.CANCELLED, error_category="cancelled"),
            (ReleaseSandbox(sandbox_id=ctx.sandbox_id, reason="cancelled"),),
        )

    if isinstance(event, CrashedOrCancelled):
        if ctx.state in (State.SUCCEEDED, State.FAILED, State.CANCELLED):
            return Step(ctx)
        return Step(
            replace(
                ctx,
                state=State.FAILED,
                error_category=event.category,
                error_message=event.message,
            ),
            (ReleaseSandbox(sandbox_id=ctx.sandbox_id, reason="failed"),),
        )

    if ctx.state == State.QUEUED and isinstance(event, StartRequested):
        return Step(replace(ctx, state=State.RUNNING), (AcquireSandbox(),))

    if ctx.state == State.RUNNING and isinstance(event, SandboxAcquired):
        return Step(
            replace(ctx, sandbox_id=event.sandbox_id),
            (PrepareStubEdit(sandbox_id=event.sandbox_id),),
        )

    if ctx.state == State.RUNNING and isinstance(event, SandboxAcquireFailed):
        return Step(
            replace(
                ctx,
                state=State.FAILED,
                error_category="sandbox-unavailable",
                error_message=event.reason,
            ),
            (ReleaseSandbox(sandbox_id=None, reason="failed"),),
        )

    if ctx.state == State.RUNNING and isinstance(event, StubEditPrepared):
        return Step(ctx, (CommitAndOpenPR(sandbox_id=ctx.sandbox_id or ""),))

    if ctx.state == State.RUNNING and isinstance(event, PROpened):
        return Step(
            replace(ctx, state=State.SUCCEEDED, pr_id=event.pr_id, pr_number=event.pr_number),
            (ReleaseSandbox(sandbox_id=ctx.sandbox_id, reason="succeeded"),),
        )

    # Unknown event for current state — no-op. Logged at the caller.
    return Step(ctx)
