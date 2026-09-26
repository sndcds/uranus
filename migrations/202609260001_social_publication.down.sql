BEGIN;

-- Stop publishers before rollback. This destroys resolved history and its
-- duplicate-content protection; export it before deliberately rolling back.
LOCK TABLE uranus.social_post_target, uranus.social_publication IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM uranus.social_publication WHERE status IN ('publishing', 'uncertain'))
        OR EXISTS (SELECT 1 FROM uranus.social_post_target WHERE status = 'publishing') THEN
        RAISE EXCEPTION 'reconcile publications before rolling back Part 6';
    END IF;
END;
$$;

DROP TABLE uranus.social_publication;
DROP FUNCTION uranus.protect_social_publication();
ALTER TABLE uranus.social_post_target DROP CONSTRAINT social_post_target_publication_key;

COMMIT;
