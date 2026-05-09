"""Pure tests for RunStateMachine. No I/O, no model, no sandbox."""

from __future__ import annotations

from app.state_machine import (
    AcquireSandbox,
    CancelRequested,
    CommitAndOpenPR,
    Context,
    CrashedOrCancelled,
    PROpened,
    PrepareStubEdit,
    ReleaseSandbox,
    SandboxAcquired,
    SandboxAcquireFailed,
    StartRequested,
    State,
    StubEditPrepared,
    step,
)


def drive(events):
    ctx = Context()
    intents = []
    for e in events:
        s = step(ctx, e)
        ctx = s.context
        intents.extend(s.intents)
    return ctx, intents


def test_happy_path_reaches_succeeded():
    ctx, intents = drive([
        StartRequested(),
        SandboxAcquired(sandbox_id="sbx-1"),
        StubEditPrepared(),
        PROpened(pr_id="pr-1", pr_number=42),
    ])
    assert ctx.state == State.SUCCEEDED
    assert ctx.pr_id == "pr-1"
    assert ctx.pr_number == 42
    assert any(isinstance(i, AcquireSandbox) for i in intents)
    assert any(isinstance(i, PrepareStubEdit) for i in intents)
    assert any(isinstance(i, CommitAndOpenPR) for i in intents)
    assert any(isinstance(i, ReleaseSandbox) and i.reason == "succeeded" for i in intents)


def test_sandbox_acquire_failure_marks_failed():
    ctx, intents = drive([
        StartRequested(),
        SandboxAcquireFailed(reason="quota exceeded"),
    ])
    assert ctx.state == State.FAILED
    assert ctx.error_category == "sandbox-unavailable"
    assert "quota" in (ctx.error_message or "")
    assert any(isinstance(i, ReleaseSandbox) and i.reason == "failed" for i in intents)


def test_cancel_mid_run_releases_sandbox():
    ctx, intents = drive([
        StartRequested(),
        SandboxAcquired(sandbox_id="sbx-9"),
        CancelRequested(),
    ])
    assert ctx.state == State.CANCELLED
    release = next(i for i in intents if isinstance(i, ReleaseSandbox))
    assert release.sandbox_id == "sbx-9"
    assert release.reason == "cancelled"


def test_crash_marks_failed_with_category():
    ctx, _ = drive([
        StartRequested(),
        SandboxAcquired(sandbox_id="sbx-2"),
        CrashedOrCancelled(category="agent-error", message="boom"),
    ])
    assert ctx.state == State.FAILED
    assert ctx.error_category == "agent-error"
    assert ctx.error_message == "boom"


def test_terminal_states_are_idempotent():
    ctx, _ = drive([
        StartRequested(),
        SandboxAcquireFailed(reason="x"),
    ])
    # Further events on a failed Run should not change state.
    s = step(ctx, CancelRequested())
    assert s.context.state == State.FAILED
    s = step(ctx, PROpened(pr_id="x", pr_number=1))
    assert s.context.state == State.FAILED


def test_unknown_event_in_current_state_is_a_noop():
    ctx = Context(state=State.RUNNING, sandbox_id="sbx-x")
    s = step(ctx, StartRequested())  # not valid in RUNNING
    assert s.context == ctx
    assert s.intents == ()
