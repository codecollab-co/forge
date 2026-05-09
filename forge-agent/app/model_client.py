"""ModelClient — the only place an LLM SDK is allowed to be imported.

Per ADR-0006, every model call in forge-agent flows through this module.
The CI lint check rejects `import anthropic` anywhere else in the codebase.

Slice 8 ships an Anthropic implementation + a recorded-fixture fake used by
tests. Multi-vendor routing and BYOK key resolution arrive in follow-ups —
the interface is shaped so they're contained changes.
"""

from __future__ import annotations

import asyncio
import os
from dataclasses import dataclass, field
from typing import Any, Literal, Protocol, Sequence

import anthropic

DEFAULT_MODEL = "claude-opus-4-7"
DEFAULT_MAX_TOKENS = 4096


@dataclass(frozen=True)
class ToolDef:
    name: str
    description: str
    input_schema: dict


@dataclass(frozen=True)
class ToolUse:
    id: str
    name: str
    input: dict


@dataclass(frozen=True)
class ToolResult:
    tool_use_id: str
    content: str
    is_error: bool = False


@dataclass(frozen=True)
class Message:
    role: Literal["user", "assistant"]
    # `content` mirrors the Anthropic shape: either a plain string or a list
    # of typed blocks ({"type": "text", "text": ...} | {"type": "tool_use", ...}
    # | {"type": "tool_result", ...}).
    content: str | list[dict]


@dataclass(frozen=True)
class ModelResponse:
    stop_reason: Literal["end_turn", "tool_use", "max_tokens", "stop_sequence", "error"]
    text_blocks: tuple[str, ...] = field(default_factory=tuple)
    tool_uses: tuple[ToolUse, ...] = field(default_factory=tuple)
    usage_input: int = 0
    usage_output: int = 0


class ModelClient(Protocol):
    async def call(
        self,
        *,
        system: str,
        messages: Sequence[Message],
        tools: Sequence[ToolDef],
        model: str = DEFAULT_MODEL,
        max_tokens: int = DEFAULT_MAX_TOKENS,
    ) -> ModelResponse: ...


class AnthropicModelClient:
    """Thin wrapper over anthropic.AsyncAnthropic.

    No retry logic here at MVP — the Runner does one retry on transient
    failure. Rate-limit and overload errors are bubbled up so the Runner
    can decide.
    """

    def __init__(self, api_key: str | None = None) -> None:
        self._client = anthropic.AsyncAnthropic(api_key=api_key or os.environ.get("ANTHROPIC_API_KEY"))

    async def call(
        self,
        *,
        system: str,
        messages: Sequence[Message],
        tools: Sequence[ToolDef],
        model: str = DEFAULT_MODEL,
        max_tokens: int = DEFAULT_MAX_TOKENS,
    ) -> ModelResponse:
        resp = await self._client.messages.create(
            model=model,
            max_tokens=max_tokens,
            system=system,
            tools=[
                {"name": t.name, "description": t.description, "input_schema": t.input_schema}
                for t in tools
            ],
            messages=[{"role": m.role, "content": m.content} for m in messages],
        )
        text_blocks: list[str] = []
        tool_uses: list[ToolUse] = []
        for block in resp.content:
            btype = getattr(block, "type", None)
            if btype == "text":
                text_blocks.append(block.text)
            elif btype == "tool_use":
                tool_uses.append(ToolUse(id=block.id, name=block.name, input=dict(block.input)))
        return ModelResponse(
            stop_reason=resp.stop_reason or "end_turn",
            text_blocks=tuple(text_blocks),
            tool_uses=tuple(tool_uses),
            usage_input=resp.usage.input_tokens,
            usage_output=resp.usage.output_tokens,
        )


@dataclass
class FakeModelClient:
    """Recorded-fixture client. Each `call()` returns the next response in
    `responses`. Use in tests to drive the agent loop deterministically."""

    responses: list[ModelResponse]
    calls: list[dict] = field(default_factory=list)

    async def call(
        self,
        *,
        system: str,
        messages: Sequence[Message],
        tools: Sequence[ToolDef],
        model: str = DEFAULT_MODEL,
        max_tokens: int = DEFAULT_MAX_TOKENS,
    ) -> ModelResponse:
        self.calls.append(
            {"system": system, "messages": list(messages), "tools": [t.name for t in tools], "model": model}
        )
        if not self.responses:
            raise RuntimeError("FakeModelClient: no more recorded responses")
        # Tiny await so it's a real coroutine.
        await asyncio.sleep(0)
        return self.responses.pop(0)


def from_env() -> ModelClient:
    return AnthropicModelClient()
