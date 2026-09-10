# Team onboarding notifications

## Architecture decision

The existing event/partner notification queries derive reminders from current
business data; they have no persistent read/dismiss lifecycle. The legacy
`message` inbox is unfinished (integer user columns in its DDL and mismatched
`*_id` queries). It cannot safely serve UUID-based system notifications. There is
no existing outbox, job runner, or migration runner. This feature adds generic
`user_notification` records and a small associated email outbox, without changing
those existing interfaces.

`OrgTeamInviteAccept` retains its row-level `FOR UPDATE` lock. It checks the stored
token and `has_joined`, activates membership, clears the token, creates a missing
`user_organization_link` with permissions zero, then writes two notifications and
two outbox jobs, all in one transaction. Existing permission links are preserved.
A failure before commit leaves the invitation usable and creates no notifications
or mail. Concurrent/replayed accepts cannot pass the consumed token check.

The unique `(user_uuid, type, event_key)` constraint provides additional dedupe.
The event key is an internal SHA-256 fingerprint of the invitation token, never
exposed by the API. No bearer token or email address is stored in notification
metadata. The unique outbox notification UUID permits one mail job per record.
The sender is read from `invited_by_user_uuid`; no other administrators receive
its notification. If that user was deleted (`ON DELETE SET NULL`), joining still
succeeds and only the member receives the welcome notification.

Types: `organization_team_invite_accepted` and `organization_team_joined`.
Metadata contains organization/member display names (no email fallback). The
frontend translates these types; mail templates use the recipient's current
`user.locale` (`de`, `en`, `da`, otherwise English). Template lookup uses the
existing `SqlGetSystemEmailTemplate`, with English fallback if a locale template
is absent. HTML substitutions are escaped and subject substitutions strip CR/LF.

## Deployment

Apply both **new** migrations once, before starting this API version. There is no
automatic migration on service startup. Use the application's database and schema:

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -v schema=uranus -f migrations/20260910_team_onboarding.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -v schema=uranus -f migrations/20260910_team_onboarding_templates.sql
```

Use the deployment system to record the applied migration. The first migration
intentionally fails if applied twice; the template inserts preserve customizations.
Do not execute historical `ddl/` snapshots as migrations. PostgreSQL must provide
`gen_random_uuid()` (built into supported PostgreSQL versions since 13).

Set the **existing** `frontend` configuration field to the trusted dashboard base
URL, for example `https://dashboard.example`. Acceptance does not trust an HTTP
Referer/Origin or a client-supplied URL to build follow-up mail links. Invalid or
missing configuration keeps mail jobs pending while membership and notifications
remain available. Deploy the API/migrations before the accompanying dashboard PR.

The inviter link is `/admin/org/{orgUuid}/member/{memberUuid}/permissions`.
The welcome link is the existing `/admin/orgs` route: individual edit/team routes
require permissions that a newly joined member may not yet have. The existing
organization-list query includes zero-permission memberships.

## Email transaction boundary and delivery tradeoff

A worker starts with the API and scans up to 20 pending jobs every 15 seconds.
Missing templates, invalid frontend configuration, or database preparation errors
leave jobs pending for the next pass. Correcting the problem also recovers pending
work after restarts. The worker reads the recipient and template, prepares the
message, then atomically changes `pending` to `sending` **before** SMTP. There is
no database transaction held open during `sendEmailWithTimeout`.

Only the worker winning that conditional update calls SMTP. A successful helper
return marks `sent`; an error/timeout marks `failed`. A crash after claiming leaves
`sending`. Both states are intentionally excluded from automatic retries. The
existing SMTP helper can continue in its goroutine after returning a timeout, and
SMTP cannot guarantee exactly-once delivery across a crash/acknowledgement loss.
This implementation favors **at most one application send attempt** over silently
resending an ambiguous message. Membership is never undone by SMTP failure.
Successful SMTP handoff does not guarantee inbox delivery.

Operations should monitor failed jobs and stale sending jobs (e.g. over 5 minutes):

```sql
SELECT uuid, status, created_at, attempted_at
FROM uranus.notification_email_outbox
WHERE status = 'failed'
   OR (status = 'sending' AND attempted_at < now() - interval '5 minutes');
```

Investigate mail-provider logs before retrying. Only after confirming no delivery
and no still-running send, an operator may explicitly reset a selected UUID to
`pending`. An uncertain delivery must not be reset automatically. This is the
remaining delivery/reliability tradeoff; the outbox is not an exactly-once SMTP
service. Logs from this worker identify job UUIDs without tokens or recipients.

## Notification API and lifecycle

All routes run behind the existing access-token JWT middleware:

- `GET /api/admin/user/notifications?status=active&offset=0`: default active records
  (read and unread, excluding dismissed), 50 per page and `has_more`. Optional
  status values: `unread`, `read`, `dismissed`.
- `PATCH /api/admin/user/notifications/:notificationUuid/read`
- `PATCH /api/admin/user/notifications/:notificationUuid/dismiss`

Responses use the existing grains API envelope. Reads and mutations constrain
`user_uuid` to the JWT authentication context, never a request-supplied user UUID.
Foreign and missing UUIDs both return 404 for mutations. Repeated actions preserve
the first timestamp. Displaying a record does not change it. Dismissal persists in
the database; no hard-delete API is exposed. Deleted recipients cascade their
notifications/mail jobs; deleted organizations cascade associated notifications;
deleted actors/targets are set null.

The permission editor keeps its server-side `ManagePermissions` check. Its member
lookup is now constrained by **both** organization and member UUID; previously it
could select the wrong organization for members of multiple teams. Permission and
invite denials now preserve their HTTP status rather than returning generic 500s.
Team-list responses add `permissions_missing`, avoiding JavaScript bigint bitmask
conversion. The frontend only shows the permission CTA to managers.

## Verification

PostgreSQL integration tests use an isolated randomly named schema and apply the
actual new migrations. The base fixture uses the existing user/member/permission
DDL and a minimal organization table, so no production database or SMTP is needed:

```sh
URANUS_TEST_DATABASE_URL='postgres://test@localhost:5432/test?sslmode=disable' go test -race ./...
go vet ./...
```

Tests cover valid, invalid, expired, no-expiry, mismatched, consumed and concurrent
accepts; transaction rollback; existing permissions; recipient dedupe; JWT ownership;
read/dismiss; permission/invite denials; the complete invite-to-permission API flow;
DE/EN/DA/fallback template rendering; escaped values; concurrent workers; SMTP failure;
recoverable preparation errors; claimed jobs after restart; and deleted inviters.
The database tests skip when `URANUS_TEST_DATABASE_URL` is absent.

## Manual verification (deployment checklist, not a claim of live delivery)

1. User A with team management rights invites existing user B from the team page.
2. B receives the original invite email and opens its acceptance link.
3. Confirm successful membership; replay the link and confirm no further work.
4. A receives the acceptance mail and sees the dashboard notification.
5. A clicks “Set permissions” and reaches B's exact permission editor.
6. A assigns rights; team status no longer reports missing permissions.
7. B receives the welcome mail and sees the welcome dashboard notification.
8. B opens Organizations and sees the joined organization even with zero rights.
9. Mark notifications read (still visible), dismiss, reload and sign in again;
   dismissed notifications remain hidden. Verify in German, English and Danish.
10. Repeat on a narrow viewport and with a failed notification API request.

Live mailbox delivery and browser verification against a deployed pair of services
require the deployment environment and real test accounts; automated tests use a
fake SMTP transport and a disposable local PostgreSQL instance.
