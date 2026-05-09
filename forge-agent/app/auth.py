"""JWT verification for tokens issued by forge-platform (RS256).

Public key supplied via JWT_PUBLIC_KEY env var, PEM-encoded with literal \\n
escapes for newlines (matches the docker-compose convention).
"""

from __future__ import annotations

import os

import jwt


class AuthError(Exception):
    pass


def _public_key() -> str:
    raw = os.environ.get("JWT_PUBLIC_KEY", "").strip()
    if not raw:
        raise AuthError("JWT_PUBLIC_KEY is empty")
    return raw.replace("\\n", "\n")


def verify(token: str) -> dict:
    try:
        return jwt.decode(
            token,
            _public_key(),
            algorithms=["RS256"],
            issuer="forge-platform",
        )
    except jwt.PyJWTError as exc:
        raise AuthError(str(exc)) from exc
