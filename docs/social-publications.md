# Social publication history and recovery (Part 6)

Part 6 extends [social accounts](social-accounts.md), [posts and targets](social-posts.md),
[ContentItems](content-items.md), [preview](social-preview.md) and
[manual publishing](social-publishing.md). The Social admin API continues to use
Markdown documentation and the existing Grains response envelope.

## Architecture

```text
social_post → social_post_target → social_publication
             current state        history of individual attempts

ContentItem → RenderedPost → fingerprint + persisted snapshot
                          → COMMIT → SocialPublisher → remote platform
                          → transaction: publication outcome + target state
```

The existing claim / publish / finish functions remain responsible for publishing.
The claim transaction checks permissions, locks the post, loads the current source,
renders each eligible target once, checks its history, claims the target and inserts
an attempt. Every attempt is committed before any remote mutation. The same
`RenderedPost` supplies the fingerprint, snapshot and publisher input. Changing an
event after claim does not change that attempt's payload.

HTTP never runs inside a database transaction. Each target's outcome is finalized
in its own short transaction; one failed target cannot undo another target's success.
The existing bounded, cancellation-detached finalization still lasts at most five
seconds. Preview creates no publication, performs no write and makes no remote call.

## Schema and identity

`social_publication` stores:

- `uuid`, `social_post_uuid`, `social_post_target_uuid`, `social_account_uuid`, `platform`;
- `content_fingerprint`, `rendered_text`, `rendered_image_url`, `rendered_image_alt`;
- `status`, `remote_post_id`, `error`;
- `started_at`, `finished_at`, `reconciled_at`, `created_at`, `updated_at`.

UUIDs use the existing UUIDv7 helper and times use `timestamptz`. There are no
revision or attempt counters: group by target and fingerprint, then order by
`created_at, uuid`. Each retry has a new UUID. A different fingerprint identifies
a different rendered revision. History rows contain neither full account objects
nor credentials. `platform` remains available even if account metadata changes.

A composite foreign key to the target verifies all three references together.
It uses `ON DELETE RESTRICT`; the target's existing post/account foreign keys
supply the remaining relationship guarantees. Target removal, post deletion and
account deletion cannot erase history. The post/target APIs return 409 for a
history-protected deletion; organization cascades are also rejected by PostgreSQL.
The existing target/account ownership protection remains in place.

A database trigger rejects history deletion, changes to snapshot/identity/start
fields, and changes to resolved outcomes. Only an unresolved row's outcome and
completion metadata can change. Rendering/account preflight failures can have
NULL fingerprint and snapshot fields because no remote payload was produced;
the failed attempt records the safe reason. An already missing account, possible
only with broken target foreign keys, is rejected before remote work and cannot
have a valid history reference.

Indexes have specific purposes:

| Index | Purpose |
| --- | --- |
| UUID primary key | Detail and reconciliation |
| Unique target/fingerprint where published | Successful-content idempotency |
| Unique target where publishing/uncertain | One unresolved attempt per target |
| Post + created_at + uuid | Post history and permission-scoped organization queries |
| Target + created_at + uuid | Target history and FK protection |
| Account | Account history/reference lookup |
| Status + created_at + uuid | Outcome filters and unresolved recovery lists |
| Created_at + uuid | Deterministic complete history ordering |

There is no standalone fingerprint index: the actual duplicate query is scoped
to a target and uses the partial unique index. The composite referenced target key
also prevents history from becoming detached through direct target reassignment.

## Fingerprint and duplicate protection

`service.SocialContentFingerprint` uses SHA-256, encoded as 64 lowercase hex
characters. The byte serialization is exactly this sequence:

1. `RenderedPost.Platform`
2. `RenderedPost.Text`
3. `RenderedPost.ImageURL`
4. `RenderedPost.ImageAlt`, or the empty string when nil or when there is no image

Each field is prefixed with its UTF-8 byte length as an unsigned 64-bit big-endian
integer. Bytes are not trimmed or Unicode-normalized. Field boundaries cannot
collide; maps, locale and struct formatting play no part. A nil and empty alt text
have the same remote meaning. Alt text without an image is not sent and does not
change the fingerprint. `RenderedPost.URL` is metadata already represented in
`Text`; Mastodon does not receive it separately. Fixed public visibility adds no
variable payload field. A future publisher that sends additional fields must
extend this definition deliberately before publishing them.

Idempotency is **social_post_target_uuid + content_fingerprint**, not global text
deduplication and not account-wide deduplication:

- A successful A blocks another A with 409 (or an individual conflict in 207).
- Changed content B can be published through the same target, including from
  `published`. An explicit language change can also change the rendered content.
- A definitive failure permits a manual retry of A with a new history row.
- A published A, then published B, then a return to A still blocks A.
- Another account/target or a distinct social post is a separate publishing
  context. Two posts referencing the same event may intentionally publish it.
- A target with history cannot be deleted and recreated to bypass its guard.
- Different organizations never share a deduplication key.

The post row lock and conditional target claim serialize competing API requests.
The partial unique indexes provide additional database protection. As in Part 5,
one unresolved target blocks the entire post's next publish request, preventing
an overlapping request from retrying targets in an active batch.

## State transitions and current state

| Publication transition | Target result |
| --- | --- |
| New attempt → publishing | publishing |
| publishing → published | published; new remote ID and published_at |
| publishing → failed | failed; safe error; prior successful ID/time retained |
| publishing → uncertain | publishing; blocked |
| uncertain → published, by reconciliation | published; supplied remote ID and confirmation time |
| uncertain → failed, by reconciliation | failed; remote ID and published_at cleared |
| abandoned publishing → published/failed, by confirmed recovery | Same reconciliation rules |

`social_post_target.status` describes the latest attempt/current workflow.
`remote_post_id` and `published_at` normally retain the last successful publication,
including while a new revision is publishing or has definitively failed.
`error` describes the latest relevant target failure. Each history row retains
its own remote ID, error and completion time. A successful attempt clears the
current target error.

The explicit **reconcile failed** repair clears the target's remote ID and time,
as it confirms that the unresolved attempt produced no remote post. Earlier
successful IDs remain available in history. `finished_at` means the time an
outcome was recorded (including uncertainty); reconciliation sets it to the time
of confirmation and also sets `reconciled_at`. A manually confirmed publication
cannot recover an unknown historical remote creation timestamp.

## History API

All routes require the existing admin JWT middleware, accepted organization
membership and `UserPermEditOrg`. No new permission bit is introduced.

| Method | Path | Result |
| --- | --- | --- |
| GET | `/api/admin/social/publications` | `data.publications`, including `[]` |
| GET | `/api/admin/social/publications/:uuid` | Publication fields in `data` |
| POST | `/api/admin/social/publications/:uuid/reconcile` | Resolved publication in `data` |

List accepts optional `org_uuid`, `social_post_uuid`, `social_post_target_uuid`
and `status` filters, combined with AND. Status is one of `publishing`,
`published`, `failed`, `uncertain`. Results are ordered by `created_at, uuid`
ascending and scoped to organizations the caller can edit. A foreign organization
filter returns `[]`; detail/reconciliation without permission returns 403.
The existing Social and comparable organization admin lists do not paginate, so
this endpoint uses the same convention. Very large histories will need a future
shared pagination convention.

Malformed identifiers/filters/payloads return 400, missing authentication 401,
missing publications 404, already resolved or conflicting recovery requests 409,
and sanitized internal database errors 500. Unknown JSON properties are rejected.
No credential, Authorization header or remote response body is returned or logged.
Publication errors are fixed safe messages and have a database limit of 1,024
characters. Existing publisher transport, SSRF and response-sanitization protections
are unchanged.

The existing `POST /api/admin/social/posts/:uuid/publish` keeps its fields and
200/207/409 semantics. Attempt results add `publication_uuid` and, when rendering
succeeded, `content_fingerprint`. A duplicate conflict includes the newly computed
fingerprint but creates no attempt and makes no remote request.

```json
{
  "target_uuid": "01994126-6680-7000-8000-000000000031",
  "social_account_uuid": "01994126-6680-7000-8000-000000000020",
  "platform": "mastodon",
  "status": "published",
  "remote_post_id": "123456789",
  "publication_uuid": "01994126-6680-7000-8000-000000000040",
  "content_fingerprint": "<64 lowercase hex characters>"
}
```

## Uncertainty and manual recovery

A timeout, cancellation, connection loss, HTTP 5xx or invalid status success
response can mean Mastodon created the post. The history row becomes `uncertain`
and the target remains `publishing`. Do not retry it before inspecting Mastodon.
Reconciliation is strictly local: **no Mastodon GET or POST** is performed.

If the operator finds the post:

```http
POST /api/admin/social/publications/<uuid>/reconcile
Content-Type: application/json

{"outcome":"published","remote_post_id":"123456789"}
```

The ID is mandatory and must satisfy the existing Mastodon bounded decimal-ID
syntax (1–64 digits). Unsupported platforms have no published-ID validator yet.
The transaction resolves the publication and target together and records
`reconciled_at`. The fingerprint now blocks duplicate publication.

Only after the operator establishes that no post was created:

```http
POST /api/admin/social/publications/<uuid>/reconcile
Content-Type: application/json

{"outcome":"failed"}
```

`remote_post_id` must be omitted, including no explicit null. The persisted error
is `operator confirmed no remote post was created`. A subsequent explicit call
to the existing publish route creates a new attempt. There is no retry endpoint.
Resolved `published`/`failed` rows cannot be reconciled again. Row locks and the
same transaction ensure that only one of two concurrent resolutions wins.

### Crash or finalization database failure

The committed `publishing` row survives if the process dies, the claim's HTTP
work never starts, or PostgreSQL becomes unavailable after a remote mutation.
It is unresolved and recoverable, even though a database outage makes persisting
`uncertain` impossible. Listing `status=publishing` exposes those rows.

**Stop/drain all active publish requests across every API replica before recovering
such a row.** A durable claim alone cannot tell whether a publisher is still
running; age is not proof. Reconciliation of `publishing` requires the additional
explicit operator assertion:

```json
{"outcome":"published","remote_post_id":"123456789","confirm_inactive":true}
```

The same assertion is required for a failed outcome from `publishing`. It is not
automatic liveness detection. Falsely asserting inactivity can race a live remote
mutation; operators must enforce the prerequisite. Ordinary completed `uncertain`
attempts do not need this assertion. A late local finalizer cannot overwrite an
already resolved publication because its update only accepts `publishing`.

## Migration and legacy recovery

Apply `202609260001_social_publication.up.sql` after the existing account, post
and `202609250003_social_publish` migrations, with publishers stopped during the
upgrade. DDL uses the same explicit `uranus` schema as earlier migrations; adapt
it consistently when deploying a different schema. Canonical installation uses
`ddl/social_publication.ddl` after the target DDL.

The migration locks targets and creates one synthetic `uncertain` publication
for every legacy `publishing` target. Its UUID equals the target UUID in the
separate publication table; its start time is the target's last update. Its error
identifies Part 5 and its unavailable rendered snapshot. Fingerprint and all
snapshot fields are NULL. No remote ID or historical content is inferred from
today's event. Reconcile these rows through the normal API after remote inspection.
A failed resolution allows a freshly rendered attempt with a real fingerprint.

Part-5 successful publications cannot acquire a trustworthy fingerprint
retroactively. Existing `published` targets without successful history, and
legacy uncertain attempts reconciled as published with NULL fingerprints, remain
blocked from republishing. This deliberately preserves Part 5's protection;
new-content republishing is supported for successes recorded with a Part-6
snapshot. An intentionally separate publication can use a distinct social post.

The down migration locks targets/history and rejects unresolved `publishing` or
`uncertain` rows, including target-only unresolved claims. With only resolved
history, rollback **destroys the history and its fingerprint protection**; export
it first. Target data is retained, including its latest state. Stop publishers
before rollback. Migration tests cover Parts 1/2/5 → Part 6 up → guarded down →
resolved down → up again, plus the canonical DDL, FK/unique/status/immutability
constraints and legacy recovery.

## Limits and scope

There is no exactly-once guarantee across PostgreSQL and a remote HTTP service.
Recovery depends on an operator's correct remote inspection and, for abandoned
claims, enforced publisher inactivity. An image URL is fingerprinted, not its
binary contents; changing bytes behind the same URL is outside this definition.
Snapshots can contain published personal data and have no retention/deletion API
in this part. Direct administrative database maintenance remains privileged.

No scheduling, worker, cron, automatic retry, queue, automation rule, event-created
trigger, remote reconciliation verification or dashboard UI is introduced.
Facebook, Instagram and Bluesky still return their existing not-implemented
publishing error. All automated publishing tests use local TLS mocks.

## Verification

```sh
export GOCACHE=/tmp/uranus-go-build
go test ./...
go vet ./...
go build ./...
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
URANUS_AUTH_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
  go test -race -count=1 ./...
```

Use an explicitly disposable PostgreSQL/PostGIS database. Without these variables,
the existing database harness skips its integration tests. Part 6 adds no skips.
