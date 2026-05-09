"""Agent tool definitions and execution.

Slice 8 ships four tools:
  - list_files(dir): list paths under a directory
  - read_file(path): read a UTF-8 file's contents
  - write_file(path, content): create or overwrite a file
  - final_answer(summary): signal that the agent is done

`run_shell` is intentionally deferred: it requires real sandbox execution
(E2B), which lands as a follow-up. Without `run_shell` the agent can read
and edit code but cannot run tests inside the loop — acceptable for MVP.

Tools act on a Workspace (the slice-8 sandbox abstraction): a virtual
filesystem of `path -> str`. Real E2B sandboxes will swap in by satisfying
the same Workspace protocol.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol

from app.model_client import ToolDef


class Workspace(Protocol):
    def list_files(self, dir: str) -> list[str]: ...
    def read_file(self, path: str) -> str | None: ...
    def write_file(self, path: str, content: str) -> None: ...
    def changed_files(self) -> list[tuple[str, str]]: ...


TOOL_DEFS: tuple[ToolDef, ...] = (
    ToolDef(
        name="list_files",
        description="List file paths under a directory in the repository. Pass an empty string for the repository root.",
        input_schema={
            "type": "object",
            "properties": {"dir": {"type": "string", "description": "Directory path; '' for root."}},
            "required": ["dir"],
        },
    ),
    ToolDef(
        name="read_file",
        description="Read a UTF-8 text file from the repository.",
        input_schema={
            "type": "object",
            "properties": {"path": {"type": "string"}},
            "required": ["path"],
        },
    ),
    ToolDef(
        name="write_file",
        description="Create or overwrite a UTF-8 text file in the repository.",
        input_schema={
            "type": "object",
            "properties": {
                "path": {"type": "string"},
                "content": {"type": "string"},
            },
            "required": ["path", "content"],
        },
    ),
    ToolDef(
        name="final_answer",
        description="Call when you are done. Provide a short summary of the changes you made for the PR description.",
        input_schema={
            "type": "object",
            "properties": {"summary": {"type": "string"}},
            "required": ["summary"],
        },
    ),
)


@dataclass(frozen=True)
class ToolOutput:
    content: str
    is_error: bool = False
    is_final: bool = False
    final_summary: str | None = None


def execute(tool_name: str, tool_input: dict, workspace: Workspace) -> ToolOutput:
    try:
        if tool_name == "list_files":
            paths = workspace.list_files(tool_input.get("dir", ""))
            return ToolOutput(content="\n".join(paths) or "(empty)")

        if tool_name == "read_file":
            path = tool_input["path"]
            blob = workspace.read_file(path)
            if blob is None:
                return ToolOutput(content=f"file not found: {path}", is_error=True)
            return ToolOutput(content=blob)

        if tool_name == "write_file":
            workspace.write_file(tool_input["path"], tool_input["content"])
            return ToolOutput(content=f"wrote {tool_input['path']}")

        if tool_name == "final_answer":
            return ToolOutput(
                content="acknowledged",
                is_final=True,
                final_summary=tool_input.get("summary", ""),
            )

        return ToolOutput(content=f"unknown tool: {tool_name}", is_error=True)
    except Exception as exc:
        return ToolOutput(content=f"tool error: {exc}", is_error=True)
