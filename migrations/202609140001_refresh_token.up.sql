BEGIN;

CREATE TABLE uranus.refresh_token (
    jti uuid PRIMARY KEY,
    user_uuid uuid NOT NULL REFERENCES uranus."user"(uuid) ON DELETE CASCADE,
    -- The first refresh token's jti identifies the session; no token secret is stored.
    family_uuid uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CONSTRAINT refresh_token_lifetime CHECK (expires_at > created_at)
);

CREATE INDEX refresh_token_user_idx ON uranus.refresh_token (user_uuid);
CREATE INDEX refresh_token_family_active_idx ON uranus.refresh_token (user_uuid, family_uuid)
    WHERE revoked_at IS NULL;
CREATE INDEX refresh_token_expires_idx ON uranus.refresh_token (expires_at);

COMMIT;
