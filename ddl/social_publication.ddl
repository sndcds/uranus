-- The composite reference preserves history and verifies all three identities.
ALTER TABLE uranus.social_post_target ADD CONSTRAINT social_post_target_publication_key
    UNIQUE (uuid, social_post_uuid, social_account_uuid);

CREATE TABLE uranus.social_publication (
    uuid uuid PRIMARY KEY,
    social_post_uuid uuid NOT NULL,
    social_post_target_uuid uuid NOT NULL,
    social_account_uuid uuid NOT NULL,
    publication_source text NOT NULL DEFAULT 'manual' CHECK (publication_source IN ('manual', 'scheduled')),
    platform text NOT NULL CHECK (platform IN ('facebook', 'instagram', 'mastodon', 'bluesky')),
    content_fingerprint text CHECK (content_fingerprint ~ '^[0-9a-f]{64}$'),
    rendered_text text,
    rendered_image_url text,
    rendered_image_alt text,
    status text NOT NULL CHECK (status IN ('publishing', 'published', 'failed', 'uncertain')),
    remote_post_id text,
    error text CHECK (char_length(error) <= 1024),
    started_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at timestamptz,
    reconciled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT social_publication_target_fkey FOREIGN KEY
        (social_post_target_uuid, social_post_uuid, social_account_uuid)
        REFERENCES uranus.social_post_target (uuid, social_post_uuid, social_account_uuid) ON DELETE RESTRICT,
    CONSTRAINT social_publication_snapshot_check CHECK (
        (content_fingerprint IS NOT NULL AND rendered_text IS NOT NULL AND rendered_image_url IS NOT NULL)
        OR (content_fingerprint IS NULL AND rendered_text IS NULL AND rendered_image_url IS NULL AND rendered_image_alt IS NULL)
    ),
    CONSTRAINT social_publication_outcome_check CHECK (
        (status = 'publishing' AND finished_at IS NULL AND remote_post_id IS NULL)
        OR (status IN ('failed', 'uncertain') AND finished_at IS NOT NULL AND remote_post_id IS NULL)
        OR (status = 'published' AND finished_at IS NOT NULL AND remote_post_id IS NOT NULL AND remote_post_id <> '')
    )
);

CREATE UNIQUE INDEX social_publication_published_idx
    ON uranus.social_publication (social_post_target_uuid, content_fingerprint) WHERE status = 'published';
CREATE UNIQUE INDEX social_publication_unresolved_idx
    ON uranus.social_publication (social_post_target_uuid) WHERE status IN ('publishing', 'uncertain');
CREATE INDEX social_publication_post_idx ON uranus.social_publication (social_post_uuid, created_at, uuid);
CREATE INDEX social_publication_target_idx ON uranus.social_publication (social_post_target_uuid, created_at, uuid);
CREATE INDEX social_publication_account_idx ON uranus.social_publication (social_account_uuid);
CREATE INDEX social_publication_status_idx ON uranus.social_publication (status, created_at, uuid);
CREATE INDEX social_publication_created_idx ON uranus.social_publication (created_at, uuid);

CREATE FUNCTION uranus.protect_social_publication() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'social publication history cannot be deleted' USING ERRCODE = '23514';
    END IF;
    IF OLD.status IN ('published', 'failed') OR
        (to_jsonb(NEW) - ARRAY['status', 'remote_post_id', 'error', 'finished_at', 'reconciled_at', 'updated_at'])
        IS DISTINCT FROM
        (to_jsonb(OLD) - ARRAY['status', 'remote_post_id', 'error', 'finished_at', 'reconciled_at', 'updated_at']) OR
        NEW.status NOT IN ('published', 'failed', 'uncertain') OR
        (OLD.status = 'uncertain' AND NEW.status = 'uncertain') THEN
        RAISE EXCEPTION 'social publication snapshot or outcome is immutable' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER social_publication_history
    BEFORE UPDATE OR DELETE ON uranus.social_publication
    FOR EACH ROW EXECUTE FUNCTION uranus.protect_social_publication();
