"""Tests for the agent tool-use loop with a recorded ModelClient."""

from __future__ import annotations

import pytest

from app.agent_loop import AgentLoopError, run_agent
from app.model_client import FakeModelClient, ModelResponse, ToolUse
from app.workspace import VirtualFilesystem


def _resp_tools(*tool_uses: ToolUse, in_tokens: int = 100, out_tokens: int = 50) -> ModelResponse:
    return ModelResponse(
        stop_reason="tool_use",
        text_blocks=(),
        tool_uses=tool_uses,
        usage_input=in_tokens,
        usage_output=out_tokens,
    )


@pytest.mark.asyncio
async def test_agent_loop_writes_file_and_returns_summary():
    vfs = VirtualFilesystem.seeded({"README.md": "# Hello\n"})
    model = FakeModelClient(
        responses=[
            _resp_tools(
                ToolUse(id="t1", name="list_files", input={"dir": ""}),
            ),
            _resp_tools(
                ToolUse(id="t2", name="read_file", input={"path": "README.md"}),
            ),
            _resp_tools(
                ToolUse(
                    id="t3",
                    name="write_file",
                    input={"path": "README.md", "content": "# Hello\n\nUpdated by agent.\n"},
                ),
            ),
            _resp_tools(
                ToolUse(id="t4", name="final_answer", input={"summary": "Added a line to README."}),
            ),
        ],
    )

    result = await run_agent(
        model=model,
        workspace=vfs,
        issue_title="Improve README",
        issue_body="Please add a sentence to the README.",
        repo_owner="alice",
        repo_name="hello",
    )

    assert "README" in result.summary
    assert vfs.read_file("README.md") == "# Hello\n\nUpdated by agent.\n"
    assert vfs.changed_files() == [("README.md", "# Hello\n\nUpdated by agent.\n")]
    assert result.iterations == 4
    assert result.input_tokens == 400


@pytest.mark.asyncio
async def test_agent_loop_fails_when_model_ends_without_final_answer():
    model = FakeModelClient(
        responses=[
            ModelResponse(stop_reason="end_turn", text_blocks=("I am done.",)),
        ],
    )
    with pytest.raises(AgentLoopError):
        await run_agent(
            model=model,
            workspace=VirtualFilesystem.seeded({}),
            issue_title="x",
            issue_body="",
            repo_owner="alice",
            repo_name="r",
        )


@pytest.mark.asyncio
async def test_agent_loop_caps_iterations():
    # Construct an infinite list-of-files loop to verify the iteration cap.
    model = FakeModelClient(
        responses=[
            _resp_tools(ToolUse(id=f"t{i}", name="list_files", input={"dir": ""}))
            for i in range(50)
        ],
    )
    with pytest.raises(AgentLoopError):
        await run_agent(
            model=model,
            workspace=VirtualFilesystem.seeded({}),
            issue_title="x",
            issue_body="",
            repo_owner="alice",
            repo_name="r",
        )
