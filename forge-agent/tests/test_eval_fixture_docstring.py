"""Eval harness — slice 8 starter fixture.

Drives the agent loop end-to-end against a deterministic fake model and
asserts the produced changes are within tolerance. Adding fixtures is
mechanical from here.
"""

from __future__ import annotations

import pytest

from app.agent_loop import run_agent
from app.model_client import FakeModelClient, ModelResponse, ToolUse
from app.workspace import VirtualFilesystem


@pytest.mark.asyncio
async def test_eval_add_docstring():
    seed = {
        "src/foo.py": "def foo(x):\n    return x + 1\n",
    }
    expected = "def foo(x):\n    \"\"\"Return x plus one.\"\"\"\n    return x + 1\n"

    model = FakeModelClient(
        responses=[
            ModelResponse(
                stop_reason="tool_use",
                tool_uses=(ToolUse(id="t1", name="read_file", input={"path": "src/foo.py"}),),
                usage_input=200, usage_output=100,
            ),
            ModelResponse(
                stop_reason="tool_use",
                tool_uses=(ToolUse(id="t2", name="write_file", input={"path": "src/foo.py", "content": expected}),),
                usage_input=200, usage_output=100,
            ),
            ModelResponse(
                stop_reason="tool_use",
                tool_uses=(ToolUse(id="t3", name="final_answer", input={"summary": "Added docstring to foo()."}),),
                usage_input=200, usage_output=100,
            ),
        ],
    )

    vfs = VirtualFilesystem.seeded(seed)
    result = await run_agent(
        model=model,
        workspace=vfs,
        issue_title="Add a docstring to foo()",
        issue_body="Please add a one-line docstring to the foo() function.",
        repo_owner="alice", repo_name="repo",
    )

    assert vfs.read_file("src/foo.py") == expected
    assert vfs.changed_files() == [("src/foo.py", expected)]
    assert "docstring" in result.summary.lower()
