"""ReviewerAgent — deep module producing a review for a Pull Request.

Pure over its inputs once a ModelClient is injected: takes a diff + context,
returns a list of ReviewComment. The Runner side (in main.py) handles fetch
+ post; this module only knows how to *form* the review.

Slice 10 ships top-level summary comments only. Inline anchoring lands in
slice 10a once we're confident the model is well-calibrated.
"""

from __future__ import annotations

from dataclasses import dataclass

from app.model_client import Message, ModelClient


# 50 KB. Above this, the diff is too large to summarise reliably; we post a
# polite "review by hand" notice instead.
MAX_DIFF_BYTES = 50_000

REVIEWER_SYSTEM_PROMPT = """You are the Forge Reviewer Agent. You review every Pull Request automatically.

Focus on:
- correctness regressions (bugs, off-by-one, wrong types, broken contracts)
- obvious security concerns (shell injection, secret leaks, missing auth)
- ambiguity that suggests the change is incomplete

Do NOT flag:
- code style / formatting
- nitpicks
- speculative refactors

Output ONE concise review summary in Markdown. Use bullet points for distinct
concerns. If nothing meaningful needs flagging, write a single sentence
acknowledging that the change looks fine. Do not invent issues to fill space.
"""

LOOKS_FINE_TEMPLATE = "_Forge Reviewer:_ no concerns flagged."

TOO_LARGE_TEMPLATE = (
    "_Forge Reviewer:_ this diff is larger than the auto-review limit "
    "({size} bytes > {limit} bytes). Please review manually."
)


@dataclass(frozen=True)
class ReviewContext:
    repo_owner: str
    repo_name: str
    pr_title: str
    pr_body: str
    base_branch: str
    head_branch: str


@dataclass(frozen=True)
class ReviewComment:
    body: str
    file: str | None = None
    line: int | None = None


async def review(
    *,
    model: ModelClient,
    diff: str,
    ctx: ReviewContext,
) -> list[ReviewComment]:
    if len(diff) > MAX_DIFF_BYTES:
        return [ReviewComment(body=TOO_LARGE_TEMPLATE.format(size=len(diff), limit=MAX_DIFF_BYTES))]

    if not diff.strip():
        # No changes to review (rare — empty diff PRs shouldn't be openable,
        # but handle defensively).
        return []

    user_prompt = (
        f"Repository: {ctx.repo_owner}/{ctx.repo_name}\n"
        f"PR: {ctx.pr_title}\n"
        f"Branches: {ctx.head_branch} → {ctx.base_branch}\n\n"
        f"PR description:\n{ctx.pr_body or '(none)'}\n\n"
        f"Diff:\n```diff\n{diff}\n```\n"
    )

    resp = await model.call(
        system=REVIEWER_SYSTEM_PROMPT,
        messages=[Message(role="user", content=user_prompt)],
        tools=[],
    )

    text = " ".join(resp.text_blocks).strip()
    if not text:
        return [ReviewComment(body=LOOKS_FINE_TEMPLATE)]

    # Prepend the agent badge so users see at a glance that this is automated.
    body = f"_Forge Reviewer:_\n\n{text}"
    return [ReviewComment(body=body)]
