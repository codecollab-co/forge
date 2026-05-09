"""VirtualFilesystemWorkspace — slice-8 sandbox execution surface.

Holds a dict of `path -> contents` seeded from the repository at HEAD.
Tools mutate this dict; at end-of-run the Runner asks for `changed_files()`
to decide what to commit.

Real E2B sandboxes will satisfy the same `Workspace` protocol from `tools`
when slice 8a lands.
"""

from __future__ import annotations

from dataclasses import dataclass, field


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
