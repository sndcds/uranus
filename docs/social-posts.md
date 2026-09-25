# Social posts and publication targets (Part 2)

[Manual publishing (Part 5)](social-publishing.md) adds the publish endpoint,
the transient `publishing` state and protection for unresolved publication claims.
The storage-only scope below describes Part 2; apply the Part 5 migration for publishing.

A social post belongs to one organization and refers to an `event`, `venue`, or
`organization` through `source_type` and `source_uuid`. The source UUID is
validated as a UUID but intentionally has no foreign key or source lookup.
Each target refers to one independent social account. Facebook and Instagram
remain separate accounts and targets.

## Storage and migration

Apply `migrations/202609250002_social_post.up.sql` after the Part 1 social-account
migration, using the existing manual SQL deployment step. There is no automatic
migration runner. Adapt the `uranus` schema consistently for custom deployments.
The former `social_post` / `social_post_destination` DDL files were unused drafts;
they are replaced by the post and target definitions. This migration expects
that those draft tables were not installed.

Both tables use application-generated UUIDv7 keys. `created_by` records the
existing authenticated user UUID and becomes null if that user is deleted.
Creation initializes audit timestamps; PUT updates the post's `updated_at`.
Retained targets keep their own UUIDs and all timestamps unchanged.

Deleting a post cascades to its targets. Deleting a referenced account is
restricted. Organization deletion removes its posts and targets before the
existing account cascade runs, using a BEFORE DELETE trigger to satisfy RESTRICT. A small database trigger also prevents moving a referenced account
to another organization. Both account operations return 409 until the targets
are removed. Target validation locks the selected account rows through commit
so account transfers or disabling cannot race the validation. The post's
organization cannot be changed through PUT.

Indexes cover the post organization, source type/UUID, target account, status,
and scheduled timestamp. The unique `(social_post_uuid, social_account_uuid)`
index also supports target lookups by post. The down migration removes the
triggers and both new tables, including their data; it leaves accounts intact.

## Admin API

All routes use the existing admin JWT middleware, cookie-origin protection, and
`UserPermEditOrg` permission with accepted organization membership.

| Method | Path | Result |
| --- | --- | --- |
| POST | `/api/admin/social/posts` | Create; 201 with post and targets |
| GET | `/api/admin/social/posts?org_uuid=<uuid>` | List; optional organization filter |
| GET | `/api/admin/social/posts/:uuid` | Post and targets; 200 |
| PUT | `/api/admin/social/posts/:uuid` | Partial update; 200 with post and targets |
| DELETE | `/api/admin/social/posts/:uuid` | Delete post and targets; 200 |

Responses use the existing Grains envelope. A single post is in `data`; lists
are in `data.posts`, including `[]` when empty. List ordering and filtering follow
Part 1 (`created_at`, then UUID), without additional pagination parameters.
Only organizations the user can edit appear in lists.

POST requires `org_uuid`, `source_type`, and `source_uuid`. `targets` is optional:

```json
{
  "org_uuid": "01994126-6680-7000-8000-000000000002",
  "source_type": "event",
  "source_uuid": "01994126-6680-7000-8000-000000000010",
  "targets": [
    {"social_account_uuid": "01994126-6680-7000-8000-000000000020"}
  ]
}
```

Each supplied account must exist, be enabled, belong to the post's organization,
and appear only once. A new target starts as `draft`. Null targets are rejected.
PUT accepts any subset of `source_type`, `source_uuid`, and `targets`.
`org_uuid` may be supplied unchanged. Omitted fields retain their values;
required fields cannot be null. Omitted targets retain every existing target;
`targets: []` removes all targets. Otherwise the array is reconciled by account
UUID: missing targets are removed, new ones inserted, retained ones untouched.
An account disabled after creation can remain on a post when targets are omitted,
but cannot be explicitly selected in a target update.

The response includes `uuid`, `org_uuid`, `source_type`, `source_uuid`,
`created_by`, `created_at`, `updated_at`, and `targets`. Each target contains
`uuid`, `social_account_uuid`, `status`, `scheduled_at`, `published_at`,
`remote_post_id`, `error`, `created_at`, and `updated_at`. Nullable values are
returned as null. No account objects, access tokens, or refresh tokens are
selected into responses. Decoder/database errors are sanitized and request
bodies or raw database errors are not logged.

Invalid input or targets return 400, missing authentication 401, insufficient
organization permissions 403, and missing posts 404. Unknown organizations have
no permission grant and return 403 on creation, matching Part 1. Database target
uniqueness violations return 409 and unexpected failures return 500. Writes,
permission checks, and reconciliation share one transaction; responses are
sent only after commit.

## Tests and remaining scope

Run `go test ./...`, `go vet ./...`, and `gofmt` for changed Go files.
PostgreSQL tests reuse Part 1's isolated-schema harness and require an explicitly
provided disposable database with PostGIS available:

```sh
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" go test -race ./...
```

Tests cover CRUD, all source types, zero/one/multiple targets, input validation,
organization permissions and isolation, duplicate/missing/disabled/foreign
accounts, target reconciliation and preservation of all publication metadata,
account deletion/transfer restrictions, cascades, migration rollback/reapply,
transaction rollback on statement/commit failures, and response/log secrecy.

Status values `draft`, `scheduled`, `published`, `failed`, and `cancelled` are
storage only. They and the publication metadata cannot be changed through these
request payloads. No rendering, ContentItem mapping, templates, preview,
publishing, scheduling, workers, external social APIs, publication history,
idempotency, or dashboard UI is introduced. Published targets do not prevent
post deletion.
