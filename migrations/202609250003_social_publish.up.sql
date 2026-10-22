BEGIN;

ALTER TABLE uranus.social_post_target DROP CONSTRAINT social_post_target_status_check;
ALTER TABLE uranus.social_post_target ADD CONSTRAINT social_post_target_status_check
    CHECK (status IN ('draft', 'scheduled', 'publishing', 'published', 'failed', 'cancelled'));

-- Preserve unresolved claims, including when an organization/post cascades.
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

COMMIT;
