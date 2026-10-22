BEGIN;

CREATE TABLE uranus.social_account (
    uuid uuid PRIMARY KEY,
    org_uuid uuid NOT NULL REFERENCES uranus.organization(uuid) ON DELETE CASCADE,
    platform text NOT NULL CHECK (platform IN ('facebook', 'instagram', 'mastodon', 'bluesky')),
    name text NOT NULL CHECK (btrim(name) <> ''),
    remote_account_id text NOT NULL CHECK (btrim(remote_account_id) <> ''),
    remote_account_name text,
    access_token text,
    refresh_token text,
    token_expires_at timestamptz,
    base_url text,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT social_account_identity_key UNIQUE (org_uuid, platform, remote_account_id)
);

COMMIT;
