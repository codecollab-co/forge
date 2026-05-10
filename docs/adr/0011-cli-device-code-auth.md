# ADR 0011 — CLI auth via device-code (RFC 8628)

**Status:** Accepted — 2026-05-10

## Context

The `forge` CLI (ADR-0010) needs to authenticate against `forge-platform`. The web flow uses SuperTokens session cookies, which a CLI cannot hold. The CLI also runs in many environments where opening a browser to a redirect URL on `localhost:<port>` is awkward (CI, remote SSH, dev containers).

## Decision

OAuth 2.0 Device Authorization Grant (RFC 8628):

1. CLI calls `POST /oauth/device/code` (no auth) → receives `{device_code, user_code, verification_uri, expires_in, interval}`.
2. CLI prints "Visit `<verification_uri>` and enter `<user_code>`" and polls `POST /oauth/device/token` every `interval` seconds with `device_code`.
3. User opens the URL in any browser, signs in if needed, enters the user code on `/device`. Web POSTs `/oauth/device/approve` with the user_code.
4. Once approved, the polling endpoint returns `{access_token, handle}`. Until then it returns 400 with `{error: "authorization_pending"}` per the RFC.
5. CLI saves `{api_url, token, handle}` to `$XDG_CONFIG_HOME/forge/credentials.json` (mode `0600`).

**Token format: long-lived RS256 JWT** signed by the same key as service-to-service tokens, `sub = user.id`, TTL = 30 days. Adds a Bearer-token middleware to `platform-api` that verifies tokens of this shape on every API endpoint, alongside the existing SuperTokens session middleware.

**Codes:** `device_code` is a 32-byte URL-safe random string (opaque). `user_code` is a 9-character `XXXX-XXXX` (alphanumeric, no ambiguous chars I, O, 0, 1).

**Storage:** `platform.device_codes` table with `(device_code TEXT PK, user_code TEXT UNIQUE, user_id UUID NULL, status TEXT [pending|approved|expired], expires_at TIMESTAMPTZ)`. Janitor deletes rows past expiry hourly.

## Consequences

- The CLI never sees the user's password or OAuth provider tokens. It only ever holds a Forge-issued JWT.
- The 30-day TTL is a deliberate trade-off: CLI users hate re-authing, and the token is single-purpose. Revocation is by deleting the user, not by individual token (CLI tokens aren't stored — the JWT is signed and self-contained). When token-level revocation matters we'll move to a DB-stored token model alongside.
- The `/device` web page is a tiny new surface. It must be served over HTTPS in production (so the user_code isn't sniffed in transit).
- The polling endpoint is unauthenticated by design, but rate-limited per device_code to prevent enumeration.

## Alternatives considered

- **PKCE (RFC 7636) with a localhost redirect.** Standard for native apps, but localhost redirects break in headless environments (SSH, CI containers). Rejected.
- **Long-lived API tokens minted from the web UI** (paste-into-CLI). Possible, but worse UX than `forge auth login` doing the dance for you. May add as a fallback for headless installs later.
- **Username + password to the CLI.** Rejected — CLIs that prompt for passwords are fragile and a phishing vector.
