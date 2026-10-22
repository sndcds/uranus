BEGIN;

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

CREATE TABLE uranus.social_post_target (
    uuid uuid PRIMARY KEY,
    social_post_uuid uuid NOT NULL REFERENCES uranus.social_post(uuid) ON DELETE CASCADE,
    social_account_uuid uuid NOT NULL REFERENCES uranus.social_account(uuid) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'scheduled', 'published', 'failed', 'cancelled')),
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

-- Account transfers must not invalidate existing target ownership. Target writes
-- in the API lock the account until commit, serializing them with transfers.
CREATE FUNCTION uranus.check_social_account_target_org() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.org_uuid IS DISTINCT FROM OLD.org_uuid AND EXISTS (
        SELECT 1 FROM uranus.social_post_target WHERE social_account_uuid = OLD.uuid
    ) THEN
        RAISE EXCEPTION 'social account is used by social post targets'
            USING ERRCODE = '23503', CONSTRAINT = 'social_post_target_social_account_uuid_fkey';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER social_account_target_org
    BEFORE UPDATE OF org_uuid ON uranus.social_account
    FOR EACH ROW EXECUTE FUNCTION uranus.check_social_account_target_org();

-- Delete posts before the organization's accounts cascade. RESTRICT is checked
-- immediately, so relying on the order of the two foreign-key cascades fails.
CREATE FUNCTION uranus.delete_org_social_posts() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM uranus.social_post WHERE org_uuid = OLD.uuid;
    RETURN OLD;
END;
$$;

CREATE TRIGGER organization_social_posts
    BEFORE DELETE ON uranus.organization
    FOR EACH ROW EXECUTE FUNCTION uranus.delete_org_social_posts();

COMMIT;
