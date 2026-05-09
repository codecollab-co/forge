"""Runner — drives the RunStateMachine for a single Run, end-to-end.

Slice 7 stub: instead of executing code inside the sandbox, the runner
acquires a sandbox, decides on a hardcoded edit (adds runs/<run-id>.md),
calls platform-api's /internal commit endpoint to create the branch +
commit, opens a PR, releases the sandbox, marks the Run succeeded.

Slice 8 will replace the hardcoded edit with the real Agent loop, leaving
this orchestration code unchanged.
"""

from __future__ import annotations

import asyncio
import datetime as dt
import logging
from dataclasses import dataclass

from app.platform_client import PlatformClient
from app.sandbox import SandboxProvider, Sandbox
from app.state_machine import (
    AcquireSandbox,
    CancelRequested,
    CommitAndOpenPR,
    Context,
    CrashedOrCancelled,
    Event,
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

logger = logging.getLogger(__name__)


@dataclass
class RunRequest:
    run_id: str
    repo_id: str
    repo_owner: str
    repo_name: str
    issue_id: str
    issue_number: int
    issue_title: str
    issue_body: str
    requested_by: str


# Hard ceiling for a single Run regardless of state. Per slice-7 design.
RUN_TIMEOUT = dt.timedelta(minutes=30)
HEARTBEAT_INTERVAL = dt.timedelta(seconds=15)


class Runner:
    def __init__(self, sandboxes: SandboxProvider, platform: PlatformClient) -> None:
        self._sandboxes = sandboxes
        self._platform = platform

    async def run(self, request: RunRequest) -> None:
        ctx = Context(state=State.QUEUED)
        sandbox: Sandbox | None = None

        async def heartbeat_loop() -> None:
            while True:
                try:
                    cancel = await self._platform.heartbeat(request.run_id)
                    if cancel:
                        await events.put(CancelRequested())
                        return
                except Exception:
                    logger.exception("heartbeat failed")
                await asyncio.sleep(HEARTBEAT_INTERVAL.total_seconds())

        events: asyncio.Queue[Event] = asyncio.Queue()
        await events.put(StartRequested())

        await self._platform.update_run_state(
            request.run_id, state="running", started_now=True
        )

        deadline = dt.datetime.now(dt.timezone.utc) + RUN_TIMEOUT
        hb_task = asyncio.create_task(heartbeat_loop())

        try:
            while ctx.state in (State.QUEUED, State.RUNNING):
                if dt.datetime.now(dt.timezone.utc) > deadline:
                    await events.put(CrashedOrCancelled(category="timeout", message="run exceeded ceiling"))

                event = await events.get()
                logger.info("run=%s event=%s state=%s", request.run_id, type(event).__name__, ctx.state)
                await self._platform.append_event(
                    request.run_id, type(event).__name__, {}
                )

                progressed = step(ctx, event)
                ctx = progressed.context

                for intent in progressed.intents:
                    if isinstance(intent, AcquireSandbox):
                        try:
                            sandbox = await self._sandboxes.acquire(run_id=request.run_id)
                            await self._platform.update_run_state(
                                request.run_id, state="running", sandbox_id=sandbox.id
                            )
                            await events.put(SandboxAcquired(sandbox_id=sandbox.id))
                        except Exception as exc:
                            logger.exception("sandbox acquire failed")
                            await events.put(SandboxAcquireFailed(reason=str(exc)))

                    elif isinstance(intent, PrepareStubEdit):
                        # Slice 7: no-op inside the sandbox; the file write
                        # happens server-side via /internal/repos/:id/commits
                        # in the next intent. Slice 8 replaces this with the
                        # real agent loop running in the sandbox.
                        await events.put(StubEditPrepared())

                    elif isinstance(intent, CommitAndOpenPR):
                        try:
                            await self._commit_and_open_pr(request, events)
                        except Exception as exc:
                            logger.exception("commit+open_pr failed")
                            await events.put(
                                CrashedOrCancelled(category="commit-failed", message=str(exc))
                            )

                    elif isinstance(intent, ReleaseSandbox):
                        if sandbox is not None:
                            try:
                                await self._sandboxes.release(sandbox)
                            except Exception:
                                logger.exception("sandbox release failed")
                        # Final state update happens in the post-loop block.
        finally:
            hb_task.cancel()
            try:
                await hb_task
            except asyncio.CancelledError:
                pass

        # Persist terminal state.
        await self._platform.update_run_state(
            request.run_id,
            state=ctx.state.value,
            error_category=ctx.error_category,
            error_message=ctx.error_message,
            pr_id=ctx.pr_id,
            finished_now=True,
        )
        logger.info("run=%s terminated state=%s", request.run_id, ctx.state)

    async def _commit_and_open_pr(self, request: RunRequest, events: asyncio.Queue) -> None:
        branch = f"forge-agent/run-{request.run_id[:8]}"
        body = (
            f"_Stub Author Agent (slice 7)._\n\n"
            f"Run `{request.run_id}` against issue "
            f"#{request.issue_number}: **{request.issue_title}**\n\n"
            f"This PR was produced without an LLM. Slice 8 replaces the stub "
            f"with the real agent loop."
        )
        file_content = (
            f"# Run {request.run_id}\n\n"
            f"- Issue: #{request.issue_number} — {request.issue_title}\n"
            f"- Created at: {dt.datetime.now(dt.timezone.utc).isoformat()}\n"
        )
        await self._platform.commit(
            request.repo_id,
            branch=branch,
            base_branch="main",
            files=[{"path": f"runs/{request.run_id}.md", "content": file_content}],
            message=f"agent: notes for run {request.run_id[:8]}",
            author={"name": "forge-agent", "email": "forge-agent@forge.local"},
        )
        pr = await self._platform.open_pr(
            request.repo_id,
            title=f"agent: handle issue #{request.issue_number}",
            body=body,
            head_branch=branch,
            base_branch="main",
            author_id=request.requested_by,
            run_id=request.run_id,
        )
        await events.put(PROpened(pr_id=pr["pr_id"], pr_number=int(pr["number"])))
