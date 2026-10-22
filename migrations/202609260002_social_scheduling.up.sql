BEGIN;

ALTER TABLE uranus.social_post_target
    ADD COLUMN publication_source text NOT NULL DEFAULT 'scheduled'
        CHECK (publication_source IN ('manual', 'scheduled')),
    ADD COLUMN publish_language text;

ALTER TABLE uranus.social_publication
    ADD COLUMN publication_source text NOT NULL DEFAULT 'manual'
        CHECK (publication_source IN ('manual', 'scheduled'));

CREATE INDEX social_post_target_due_idx
    ON uranus.social_post_target (scheduled_at, created_at, uuid) WHERE status = 'scheduled';

COMMIT;
