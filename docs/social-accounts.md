# Social accounts (Part 1)

One row in `uranus.social_account` represents one independent connection owned
by an organization. Supported `platform` values are `facebook`, `instagram`,
`mastodon`, and `bluesky` (text with a CHECK constraint).

Facebook and Instagram have their own identities and credentials. Instagram
does not reference a Facebook Page or any shared Meta credential. Its
`remote_account_id` is the Instagram Professional Account ID; Facebook uses the
Page ID, Mastodon the account ID, and Bluesky the DID. `remote_account_name` is
optional display metadata. `base_url` is nullable, normally null for Facebook and
Instagram, and used for the Mastodon server. No platform APIs are called.

## Storage and migration

Apply `migrations/202609250001_social_account.up.sql` once through the deployment
SQL migration step before starting the updated API. The repository has no
automatic migration runner. The migration uses the `uranus` schema; custom
deployments must adapt it consistently with `db_schema`.

The UUID primary key is generated with the existing UUIDv7 helper. The
`org_uuid` foreign key references `organization(uuid)` and cascades deletion.
Required fields are `org_uuid`, `platform`, `name`, and `remote_account_id`.
The unique constraint on `(org_uuid, platform, remote_account_id)` permits
the same identity in different organizations or platforms. This also applies
to Mastodon: Part 1 does not include `base_url` in the identity constraint.
`enabled` defaults to true. `token_expires_at`, `created_at`, and `updated_at`
use `timestamptz`, as in the existing social DDL and refresh-token migration.
Creation sets both audit timestamps; API updates set `updated_at` atomically.

The down migration drops the table, including all saved connections and
credentials. It is a deliberate destructive rollback, not a startup operation.

## Admin API

All routes use the existing admin JWT middleware, including cookie-origin
protection. Operations require an accepted organization membership and the
existing `UserPermEditOrg` permission. Organization administrators already have
this permission. List returns only accounts for organizations with this grant;
an inaccessible organization filter returns an empty array.

| Method | Path | Result |
| --- | --- | --- |
| GET | `/api/admin/social/accounts?org_uuid=<uuid>` | Metadata list; filter optional |
| POST | `/api/admin/social/accounts` | Create; 201 with metadata |
| GET | `/api/admin/social/accounts/:uuid` | Metadata; 200 |
| PUT | `/api/admin/social/accounts/:uuid` | Partial update; 200 with metadata |
| DELETE | `/api/admin/social/accounts/:uuid` | Hard delete, including credentials; 200 |

Responses use the existing Grains envelope. Single-account metadata is in
`data`; lists are in `data.accounts`, including `[]` when empty. Metadata contains
`uuid`, `org_uuid`, `platform`, `name`, `remote_account_id`,
`remote_account_name`, `base_url`, `enabled`, `token_expires_at`,
`has_access_token`, `has_refresh_token`, `created_at`, and `updated_at`.
Invalid input returns 400, unauthenticated requests 401, insufficient organization
permissions 403, missing accounts 404, duplicate identities 409, and unexpected
database failures 500. Unknown organizations cannot have an authorization grant
and therefore return 403 on creation.

POST accepts those configuration fields plus `access_token` and `refresh_token`,
but not server-generated UUID/audit fields. For example:

```json
{
  "org_uuid": "01994126-6680-7000-8000-000000000002",
  "platform": "instagram",
  "name": "@kulturbytes",
  "remote_account_id": "instagram-professional-account-id",
  "remote_account_name": "kulturbytes",
  "access_token": "<credential>",
  "base_url": null,
  "enabled": true
}
```

PUT accepts any subset of these fields. Omitted fields, particularly credentials,
retain their current values. Explicit `null` or `""` clears an access/refresh
token. Nullable metadata can be cleared with `null`; required fields and
`enabled` cannot be null. Token timestamps use RFC3339. Organization transfers
require permission in both organizations. Writes and permission checks use the
existing `WithTransaction` / `ApiTxError` infrastructure; row locks serialize
changes of account ownership with reads, updates, and deletes.

## Credentials

`access_token` and `refresh_token` are write-only secrets stored directly as
nullable text on the account, as requested for Part 1. There is no application
encryption layer or separate credential table. Database access, backups, and
database statement/parameter logging must therefore be treated as sensitive.
Do not enable SQL parameter tracing for this connection pool.

API response types contain no credentials. Queries return presence booleans
instead of retrieving secret values, including INSERT/UPDATE RETURNING.
The request credential type also redacts accidental JSON or formatted output.
Handlers never log request bodies, SQL arguments, or original database errors.
Decoder and database failures return fixed messages because raw error details
can contain submitted values or entire failing rows. Responses are emitted only
after a successful commit.

## Tests and scope

`go test ./...` runs validation, authentication, cookie-origin protection,
write-only serialization and error-sanitization tests.

PostgreSQL tests require an explicitly supplied disposable database with PostGIS
available and permission to create schemas/extensions:

```sh
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" go test -race ./...
```

Each test uses an isolated schema, the existing organization/user table
definitions, the existing organization permission query, and the actual up/down
migrations. Tests cover all four platforms, independent Facebook/Instagram
accounts, CRUD, identity constraints, organization isolation and transfers,
secret storage/preservation/replacement/clearing, response/log secrecy,
statement/commit rollback, cascading deletion, and migration roundtrips.

Part 2 remains separate: OAuth, external platform requests, token refresh,
connection checks, rendering, publishing, templates, content integration,
publication history, scheduling, workers/retries, and dashboard UI.
