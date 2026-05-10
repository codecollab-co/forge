"""The agent tool-use loop.

Pure-ish: takes a Workspace + ModelClient + RunRequest, returns either a
final summary (with side effects already applied to the Workspace) or
raises. No HTTP, no DB, no sandbox lifecycle — those belong to the Runner.
"""

from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass
from typing import Awaitable, Callable

from app.model_client import (
    Message,
    ModelClient,
    ModelResponse,
    ToolUse,
)
from app.tools import TOOL_DEFS, Workspace, execute as execute_tool

EventSink = Callable[[str, dict], Awaitable[None]]


async def _noop_sink(_t: str, _p: dict) -> None: ...

logger = logging.getLogger(__name__)

MAX_ITERATIONS = 20

SYSTEM_PROMPT = """You are an autonomous coding agent on Forge, an AI-native Git host.
You have been assigned to an Issue on a Repository. Your goal is to read the
relevant code, decide on a small, focused change that addresses the Issue,
and apply it via the available tools.

Rules:
- Make the smallest correct change. Do not refactor unrelated code.
- Edit files only inside the Repository.
- After meaningful changes, run the project's tests via `run_shell` if a
  test command is obvious (e.g. `pytest -q`, `go test ./...`, `npm test`).
  If tests fail, fix the regression before calling final_answer.
- Use `run_shell` for tests, formatters, linters, package managers, or
  inspecting `git status`. Don't use it for anything destructive (rm -rf,
  network exfiltration, etc.) — the sandbox enforces this anyway.
- Call `final_answer` exactly once when you are done. Provide a one-paragraph
  summary suitable for a PR description.
- If you cannot make a useful change, still call `final_answer` with an
  honest summary explaining why; do not loop.
"""


@dataclass(frozen=True)
class AgentResult:
    summary: str
    iterations: int
    input_tokens: int
    output_tokens: int


class AgentLoopError(Exception):
    pass


async def run_agent(
    *,
    model: ModelClient,
    workspace: Workspace,
    issue_title: str,
    issue_body: str,
    repo_owner: str,
    repo_name: str,
    event_sink: EventSink = _noop_sink,
) -> AgentResult:
    initial_listing = "\n".join(workspace.list_files("")[:200]) or "(empty repository)"
    user_msg = (
        f"Repository: {repo_owner}/{repo_name}\n\n"
        f"Issue title: {issue_title}\n\n"
        f"Issue body:\n{issue_body or '(no body)'}\n\n"
        f"File listing (truncated):\n{initial_listing}"
    )
    messages: list[Message] = [Message(role="user", content=user_msg)]

    total_in = 0
    total_out = 0

    for iteration in range(1, MAX_ITERATIONS + 1):
        resp = await model.call(
            system=SYSTEM_PROMPT,
            messages=messages,
            tools=list(TOOL_DEFS),
        )
        total_in += resp.usage_input
        total_out += resp.usage_output

        for text in resp.text_blocks:
            await event_sink("model.thought", {"text": text})

        if resp.stop_reason == "error":
            raise AgentLoopError("model returned an error stop_reason")

        if resp.stop_reason == "end_turn" and not resp.tool_uses:
            # Model wrote text but didn't call final_answer — treat as failure
            # rather than guess. Slice 8a may add a recovery prompt.
            text = " ".join(resp.text_blocks)
            raise AgentLoopError(f"model ended without final_answer: {text[:200]}")

        # Append the assistant turn (mixed text + tool_use blocks) to history.
        assistant_blocks: list[dict] = [{"type": "text", "text": t} for t in resp.text_blocks]
        for tu in resp.tool_uses:
            assistant_blocks.append(
                {"type": "tool_use", "id": tu.id, "name": tu.name, "input": tu.input}
            )
        messages.append(Message(role="assistant", content=assistant_blocks))

        tool_result_blocks: list[dict] = []
        final_summary: str | None = None
        for tu in resp.tool_uses:
            await event_sink("tool.use", {"name": tu.name, "input": tu.input, "id": tu.id})
            # Tool execution may hit a network-backed Workspace (E2B). Run it
            # in a thread so the event loop + heartbeat keep ticking.
            out = await asyncio.to_thread(execute_tool, tu.name, tu.input, workspace)
            await event_sink(
                "tool.result",
                {"id": tu.id, "name": tu.name, "is_error": out.is_error,
                 "content": (out.content[:500] + "…") if len(out.content) > 500 else out.content},
            )
            tool_result_blocks.append(
                {
                    "type": "tool_result",
                    "tool_use_id": tu.id,
                    "content": out.content,
                    "is_error": out.is_error,
                }
            )
            if out.is_final:
                final_summary = out.final_summary or ""

        messages.append(Message(role="user", content=tool_result_blocks))

        if final_summary is not None:
            return AgentResult(
                summary=final_summary,
                iterations=iteration,
                input_tokens=total_in,
                output_tokens=total_out,
            )

    raise AgentLoopError(f"agent loop exceeded {MAX_ITERATIONS} iterations without final_answer")


def text_only_diff(seed: dict[str, str], current: dict[str, str]) -> list[tuple[str, str]]:
    """Returns (path, content) for every file that's new or changed vs. seed."""
    out: list[tuple[str, str]] = []
    for path, content in current.items():
        if seed.get(path) != content:
            out.append((path, content))
    return out
