-- Apply once with psql -v ON_ERROR_STOP=1 -v schema=uranus -f this-file.
-- No migration runner exists in this repository. Do not apply DDL snapshots.
BEGIN;
SET LOCAL search_path TO :"schema", public;

CREATE TABLE user_notification (
    uuid uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_uuid uuid NOT NULL REFERENCES "user"(uuid) ON DELETE CASCADE,
    type text NOT NULL,
    event_key text NOT NULL,
    organization_uuid uuid REFERENCES organization(uuid) ON DELETE CASCADE,
    actor_user_uuid uuid REFERENCES "user"(uuid) ON DELETE SET NULL,
    target_user_uuid uuid REFERENCES "user"(uuid) ON DELETE SET NULL,
    action_url text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at timestamptz,
    dismissed_at timestamptz,
    UNIQUE (user_uuid, type, event_key),
    CHECK (action_url IS NULL OR (action_url LIKE '/admin/%' AND action_url NOT LIKE '%\\%'))
);
CREATE INDEX user_notification_visible_idx ON user_notification (user_uuid, created_at DESC, uuid DESC) WHERE dismissed_at IS NULL;
CREATE INDEX user_notification_unread_idx ON user_notification (user_uuid, created_at DESC) WHERE read_at IS NULL AND dismissed_at IS NULL;

CREATE TABLE notification_email_outbox (
    uuid uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_uuid uuid NOT NULL UNIQUE REFERENCES user_notification(uuid) ON DELETE CASCADE,
    template_context text NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    attempted_at timestamptz,
    sent_at timestamptz
);
CREATE INDEX notification_email_outbox_pending_idx ON notification_email_outbox (created_at) WHERE status = 'pending';
COMMIT;
