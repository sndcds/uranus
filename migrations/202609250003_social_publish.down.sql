BEGIN;

-- Stop API publishers before rollback. Never turn an uncertain remote success
-- into an eligible draft/failed target. Reconcile unresolved claims first.
LOCK TABLE uranus.social_post_target IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM uranus.social_post_target WHERE status = 'publishing') THEN
        RAISE EXCEPTION 'reconcile publishing targets before rolling back Part 5';
    END IF;
END;
$$;

DROP TRIGGER social_publish_claim_delete ON uranus.social_post_target;
DROP FUNCTION uranus.protect_social_publish_claim();
ALTER TABLE uranus.social_post_target DROP CONSTRAINT social_post_target_status_check;
ALTER TABLE uranus.social_post_target ADD CONSTRAINT social_post_target_status_check
    CHECK (status IN ('draft', 'scheduled', 'published', 'failed', 'cancelled'));

COMMIT;
