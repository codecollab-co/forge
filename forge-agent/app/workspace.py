"""VirtualFilesystem — in-process workspace used in tests and as the
local-dev default when E2B_API_KEY is empty.

Holds a dict of `path -> contents` seeded from the repository at HEAD.
Tools mutate this dict; at end-of-run the Runner asks for `changed_files()`
to decide what to commit.

`run_shell` is intentionally unavailable here — the in-memory workspace
has no shell. Returns an exit-code-1 ShellResult with a clear error so
the agent's tool-result block is informative rather than mysterious.

The real E2B-backed workspace lives in app/e2b_workspace.py.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from app.tools import ShellResult


@dataclass
class VirtualFilesystem:
    files: dict[str, str] = field(default_factory=dict)
    _seed: dict[str, str] = field(default_factory=dict)

    @classmethod
    def seeded(cls, seed: dict[str, str]) -> "VirtualFilesystem":
        return cls(files=dict(seed), _seed=dict(seed))

    def list_files(self, dir: str) -> list[str]:
        prefix = "" if dir in ("", ".", "/") else dir.rstrip("/") + "/"
        return sorted(p for p in self.files if p.startswith(prefix))

    def read_file(self, path: str) -> str | None:
        return self.files.get(path)

    def write_file(self, path: str, content: str) -> None:
        self.files[path] = content

    def changed_files(self) -> list[tuple[str, str]]:
        out: list[tuple[str, str]] = []
        for path, content in self.files.items():
            if self._seed.get(path) != content:
                out.append((path, content))
        return out

    def run_shell(self, command: str, timeout_seconds: int = 60) -> ShellResult:
        return ShellResult(
            stdout="",
            stderr="run_shell is not available in the in-memory workspace; set E2B_API_KEY",
            exit_code=1,
        )
