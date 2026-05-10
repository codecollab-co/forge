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

from app.agent_loop import AgentLoopError, run_agent
from app.e2b_workspace import E2BWorkspace
from app.model_client import ModelClient, from_env as model_from_env
from app.platform_client import PlatformClient
from app.run_pubsub import RunEvent, RunPubSub
from app.sandbox import SandboxProvider, Sandbox
from app.workspace import VirtualFilesystem
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
    def __init__(
        self,
        sandboxes: SandboxProvider,
        platform: PlatformClient,
        model: ModelClient | None = None,
        pubsub: RunPubSub | None = None,
    ) -> None:
        self._sandboxes = sandboxes
        self._platform = platform
        self._model = model
        self._pubsub = pubsub

    async def _emit(self, run_id: str, event_type: str, payload: dict) -> None:
        """Persist (via platform-api) and broadcast (via pubsub) a Run event."""
        try:
            await self._platform.append_event(run_id, event_type, payload)
        except Exception:
            logger.exception("append_event failed type=%s", event_type)
        if self._pubsub is not None:
            await self._pubsub.publish(run_id, RunEvent(type=event_type, payload=payload))

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
                await self._emit(request.run_id, "run.state_event", {"event": type(event).__name__, "state": ctx.state.value})

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
        await self._emit(
            request.run_id,
            "run.terminal",
            {"state": ctx.state.value, "error_category": ctx.error_category},
        )
        if self._pubsub is not None:
            await self._pubsub.close(request.run_id)
        logger.info("run=%s terminated state=%s", request.run_id, ctx.state)

    async def _commit_and_open_pr(self, request: RunRequest, events: asyncio.Queue) -> None:
        # Slice 8: real agent loop.
        # 1. Snapshot the repository at HEAD into a virtual filesystem.
        # 2. Drive the agent loop; tools mutate the VFS.
        # 3. Commit changed files; open the PR.
        snapshot = await self._platform.snapshot(request.repo_id)
        seed = {f["path"]: f["content"] for f in snapshot["files"]}
        # Pick the right workspace for the sandbox backend.
        if sandbox is not None and sandbox.backend == "e2b" and sandbox.handle is not None:
            workspace = E2BWorkspace(sandbox.handle)
            await asyncio.to_thread(workspace.seed, seed)
        else:
            workspace = VirtualFilesystem.seeded(seed)
        vfs = workspace  # alias used downstream

        model = self._model or model_from_env()

        async def sink(event_type: str, payload: dict) -> None:
            await self._emit(request.run_id, event_type, payload)

        try:
            agent_result = await run_agent(
                model=model,
                workspace=vfs,
                issue_title=request.issue_title,
                issue_body=request.issue_body,
                repo_owner=request.repo_owner,
                repo_name=request.repo_name,
                event_sink=sink,
            )
        except AgentLoopError as exc:
            await self._emit(request.run_id, "agent.loop_failed", {"reason": str(exc)})
            raise

        await self._emit(
            request.run_id,
            "agent.completed",
            {
                "iterations": agent_result.iterations,
                "input_tokens": agent_result.input_tokens,
                "output_tokens": agent_result.output_tokens,
            },
        )

        changed = await asyncio.to_thread(vfs.changed_files)
        if not changed:
            # Agent finished without changing anything. Treat as failure so
            # the user sees that no PR was opened.
            await events.put(
                CrashedOrCancelled(
                    category="agent-no-changes",
                    message="agent finished without proposing any file changes",
                )
            )
            return

        branch = f"forge-agent/run-{request.run_id[:8]}"
        await self._platform.commit(
            request.repo_id,
            branch=branch,
            base_branch=snapshot.get("ref") or "main",
            files=[{"path": p, "content": c} for p, c in changed],
            message=f"agent: address issue #{request.issue_number}",
            author={"name": "forge-agent", "email": "forge-agent@forge.local"},
        )

        pr_body = (
            f"_Authored by Forge Agent for run `{request.run_id[:8]}`._\n\n"
            f"**Summary**\n\n{agent_result.summary}\n\n"
            f"---\nClosed by run on issue #{request.issue_number}: {request.issue_title}"
        )
        pr = await self._platform.open_pr(
            request.repo_id,
            title=f"agent: {request.issue_title[:60]}",
            body=pr_body,
            head_branch=branch,
            base_branch=snapshot.get("ref") or "main",
            author_id=request.requested_by,
            run_id=request.run_id,
        )
        await events.put(PROpened(pr_id=pr["pr_id"], pr_number=int(pr["number"])))
