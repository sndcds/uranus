CREATE TABLE uranus.social_post (
    uuid uuid PRIMARY KEY,
    org_uuid uuid NOT NULL REFERENCES uranus.organization(uuid) ON DELETE CASCADE,
    source_type text NOT NULL CHECK (source_type IN ('event', 'venue', 'organization')),
    source_uuid uuid NOT NULL,
    created_by uuid REFERENCES uranus."user"(uuid) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX social_post_org_idx ON uranus.social_post (org_uuid);
CREATE INDEX social_post_source_idx ON uranus.social_post (source_type, source_uuid);
