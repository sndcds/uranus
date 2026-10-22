BEGIN;
DROP TRIGGER organization_social_posts ON uranus.organization;
DROP FUNCTION uranus.delete_org_social_posts();
DROP TRIGGER social_account_target_org ON uranus.social_account;
DROP FUNCTION uranus.check_social_account_target_org();
DROP TABLE uranus.social_post_target;
DROP TABLE uranus.social_post;
COMMIT;
