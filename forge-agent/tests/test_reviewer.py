"""Tests for the ReviewerAgent. No model API calls — uses FakeModelClient."""

from __future__ import annotations

import pytest

from app.model_client import FakeModelClient, ModelResponse
from app.reviewer import (
    LOOKS_FINE_TEMPLATE,
    MAX_DIFF_BYTES,
    TOO_LARGE_TEMPLATE,
    ReviewComment,
    ReviewContext,
    review,
)


def _ctx() -> ReviewContext:
    return ReviewContext(
        repo_owner="alice",
        repo_name="repo",
        pr_title="Fix off-by-one in slicing",
        pr_body="closes #4",
        base_branch="main",
        head_branch="fix/slice",
    )


@pytest.mark.asyncio
async def test_review_returns_summary_comment_from_model_text():
    model = FakeModelClient(
        responses=[
            ModelResponse(
                stop_reason="end_turn",
                text_blocks=("- Indexing change looks correct, but consider the empty-list case.",),
                usage_input=1, usage_output=1,
            ),
        ],
    )
    diff = "--- a/x.py\n+++ b/x.py\n@@\n-arr[1:]\n+arr[:1]\n"
    out = await review(model=model, diff=diff, ctx=_ctx())
    assert len(out) == 1
    assert "Forge Reviewer" in out[0].body
    assert "empty-list" in out[0].body
    assert out[0].file is None  # top-level only at MVP


@pytest.mark.asyncio
async def test_review_falls_back_to_looks_fine_on_empty_response():
    model = FakeModelClient(
        responses=[ModelResponse(stop_reason="end_turn", text_blocks=("",))],
    )
    out = await review(model=model, diff="diff", ctx=_ctx())
    assert out == [ReviewComment(body=LOOKS_FINE_TEMPLATE)]


@pytest.mark.asyncio
async def test_review_skips_oversized_diff_without_calling_model():
    model = FakeModelClient(responses=[])  # would raise if called
    big = "+" * (MAX_DIFF_BYTES + 1)
    out = await review(model=model, diff=big, ctx=_ctx())
    assert len(out) == 1
    assert "larger than" in out[0].body
    assert TOO_LARGE_TEMPLATE.format(size=len(big), limit=MAX_DIFF_BYTES) == out[0].body


@pytest.mark.asyncio
async def test_review_returns_no_comments_for_empty_diff():
    model = FakeModelClient(responses=[])
    out = await review(model=model, diff="", ctx=_ctx())
    assert out == []
