"""E2BWorkspace — Workspace protocol backed by a real E2B sandbox.

The agent's tools (read_file / list_files / write_file) act on files inside
the sandbox at /repo/. After the agent finishes, changed_files() runs
`git diff --name-only HEAD` against the seeded baseline so the runner knows
what to commit.

All public methods are sync. Runner calls these via asyncio.to_thread so
the event loop and heartbeat keep ticking during E2B network I/O.
"""

from __future__ import annotations

import logging
import shlex
from typing import Any

logger = logging.getLogger(__name__)

WORK_DIR = "/repo"


class E2BWorkspace:
    """Backed by an e2b.Sandbox handle (sync API)."""

    def __init__(self, handle: Any) -> None:
        self._sb = handle  # e2b.Sandbox

    # ---- one-shot setup ---------------------------------------------------

    def seed(self, files: dict[str, str]) -> None:
        """Populate the sandbox with the repo snapshot, then `git init` so
        we can diff later."""
        self._run(f"rm -rf {WORK_DIR} && mkdir -p {WORK_DIR}")
        for path, content in files.items():
            full = f"{WORK_DIR}/{path}"
            parent = full.rsplit("/", 1)[0]
            self._run(f"mkdir -p {shlex.quote(parent)}")
            self._sb.files.write(full, content)
        self._run(
            f"cd {WORK_DIR} && git init -q -b main && "
            "git config user.name 'forge-agent' && "
            "git config user.email 'forge-agent@forge.local' && "
            "git add -A && git commit -q -m seed --allow-empty"
        )

    # ---- Workspace protocol (sync) ----------------------------------------

    def list_files(self, dir: str) -> list[str]:
        target = WORK_DIR if dir in ("", ".", "/") else f"{WORK_DIR}/{dir.rstrip('/')}"
        result = self._run(
            f"cd {WORK_DIR} && find {shlex.quote(target)} -type f "
            "-not -path '*/.git/*' "
            f"| sed 's|^{WORK_DIR}/||' | sort"
        )
        return [p for p in result.splitlines() if p]

    def read_file(self, path: str) -> str | None:
        try:
            return self._sb.files.read(f"{WORK_DIR}/{path}")
        except Exception:
            return None

    def write_file(self, path: str, content: str) -> None:
        full = f"{WORK_DIR}/{path}"
        parent = full.rsplit("/", 1)[0]
        self._run(f"mkdir -p {shlex.quote(parent)}")
        self._sb.files.write(full, content)

    # ---- end-of-run -------------------------------------------------------

    def changed_files(self) -> list[tuple[str, str]]:
        names = self._run(
            f"cd {WORK_DIR} && git add -A && git diff --cached --name-only"
        ).splitlines()
        out: list[tuple[str, str]] = []
        for name in names:
            name = name.strip()
            if not name:
                continue
            content = self.read_file(name)
            if content is not None:
                out.append((name, content))
        return out

    # ---- helper -----------------------------------------------------------

    def _run(self, cmd: str) -> str:
        result = self._sb.commands.run(cmd, timeout=120)
        if result.exit_code != 0:
            logger.warning("sandbox cmd nonzero exit: %s\nstderr: %s", cmd, result.stderr)
        return result.stdout or ""
