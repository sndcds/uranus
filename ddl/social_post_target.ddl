CREATE TABLE uranus.social_post_target (
    uuid uuid PRIMARY KEY,
    social_post_uuid uuid NOT NULL REFERENCES uranus.social_post(uuid) ON DELETE CASCADE,
    social_account_uuid uuid NOT NULL REFERENCES uranus.social_account(uuid) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'scheduled', 'publishing', 'published', 'failed', 'cancelled')),
    scheduled_at timestamptz,
    published_at timestamptz,
    remote_post_id text,
    error text,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT social_post_target_account_key UNIQUE (social_post_uuid, social_account_uuid)
);

-- The unique index also covers lookups by social_post_uuid.
CREATE INDEX social_post_target_account_idx ON uranus.social_post_target (social_account_uuid);
CREATE INDEX social_post_target_status_idx ON uranus.social_post_target (status);
CREATE INDEX social_post_target_scheduled_idx ON uranus.social_post_target (scheduled_at);

CREATE FUNCTION uranus.protect_social_publish_claim() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'publishing' THEN
        RAISE EXCEPTION 'social target has an unresolved publication'
            USING ERRCODE = '23514';
    END IF;
    RETURN OLD;
END;
$$;

CREATE TRIGGER social_publish_claim_delete
    BEFORE DELETE ON uranus.social_post_target
    FOR EACH ROW EXECUTE FUNCTION uranus.protect_social_publish_claim();
