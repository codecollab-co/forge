-- platform.device_codes — RFC 8628 device-code OAuth (ADR-0011).
--
-- The CLI requests a device_code+user_code pair. The user enters the
-- user_code on the web /device page; that approves the row. The CLI
-- polls /oauth/device/token until the row's status is 'approved' and
-- receives a long-lived RS256 JWT.

CREATE TABLE IF NOT EXISTS platform.device_codes (
    device_code TEXT PRIMARY KEY,
    user_code   TEXT NOT NULL UNIQUE,
    user_id     UUID REFERENCES platform.users(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'approved', 'expired')),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS device_codes_user_code_idx ON platform.device_codes (user_code);
