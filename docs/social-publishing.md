# Manual social publishing (Part 5)

Part 5 builds on [social accounts](social-accounts.md), [posts and targets](social-posts.md),
[ContentItem loading](content-items.md) and [rendering/preview](social-preview.md).

## Architecture

`ContentItem → SocialRenderer → RenderedPost → SocialPublisher → remote platform`

Rendering remains a pure transformation. Preview and publish share
`socialRenderContent` and `renderSocialTarget`, including `NewSocialRenderer`,
source-language fallback, explicit `?lang` handling and image selection.
`RenderedPost.ImageAlt` carries the selected image's stored alternative text;
absent alternative text remains absent. Publishers do not generate, truncate or
re-render text, discover instance text limits, load sources or access PostgreSQL.
An instance rejecting the exact rendered text produces a publication error.

`NewSocialPublisher(platform, client, imageBaseURL)` selects the service.
Mastodon supports text and one optional image. Facebook, Instagram and Bluesky
are recognized and return `publishing is not implemented for platform <platform>`.
Unknown names return `unsupported platform`. No other platform is marked published.
The existing `remote_post_id` is sufficient: no remote URL column or publication
history table is introduced. Mastodon's response URL is not returned or stored.

## Migration and endpoint

Apply `migrations/202609250003_social_publish.up.sql` after Parts 1 and 2 using
the existing manual SQL deployment step. The canonical target DDL also includes
`publishing` and a trigger protecting deletion of unresolved claims, including
post/organization cascades. Stop publishers before rollback. The down migration
locks the target table and refuses to run while any target is `publishing`;
it never resets an uncertain publication to a publishable state.

`POST /api/admin/social/posts/:uuid/publish[?lang=de|da|en]`

No body is required. The existing admin JWT middleware and cookie-origin
protection apply. The user needs accepted membership and `UserPermEditOrg` in
the post's organization, exactly as for preview and post editing. No new global
permission bit is introduced. The endpoint is synchronous and processes only
the stored targets, in their existing created-at/UUID order.

Responses retain the Grains envelope (`response_type: admin-publish-social-post`).
The `data` property contains, for example:

```json
{
  "post_uuid": "01994126-6680-7000-8000-000000000030",
  "results": [
    {
      "target_uuid": "01994126-6680-7000-8000-000000000031",
      "social_account_uuid": "01994126-6680-7000-8000-000000000020",
      "platform": "mastodon",
      "status": "published",
      "remote_post_id": "123456789"
    }
  ]
}
```

An unsuccessful target has `error`; an absent remote ID is omitted. Conflict
results retain the target's existing status and remote ID. Response errors for
conflicts do not replace previously stored publication errors.

| HTTP status | Meaning |
| --- | --- |
| 200 | Every target was published successfully |
| 207 | At least one target was attempted, with failures, unresolved outcomes or other target conflicts; inspect every result |
| 409 | No target can be claimed; each target result explains the conflict |
| 400 | Invalid UUID or no targets |
| 401 / 403 | Missing authentication / organization permission or accepted membership, or cookie-origin rejection |
| 404 | Missing post, or source missing from the post's organization |
| 500 | Database failure; if publication has started, results include earlier outcomes and local persistence failures |

Shared source/permission/claim errors occur before any remote call and use the
existing error envelope without results. Invalid accounts or rendering failures
are isolated per target, recorded as `failed`, and included in a 207 response.

## State and concurrency

```text
draft  ─┐
        ├── publishing ── success ────────────── published
failed ─┘             ├── definite failure ──── failed
                      └── uncertain outcome ── publishing (blocked)
```

`published`, `scheduled` and `cancelled` cannot be directly published.
`failed` permits another explicit manual POST, after correcting the error.
There is no automatic retry and no `/retry` endpoint. Published targets produce
individual conflicts and no remote call; other draft/failed targets can still
be processed. Thus a mixed-success response followed by another explicit request
cannot send a published target again.

The API locks the post only in a short preparation transaction, checks the
organization grant, reads source/account metadata and renders the eligible
targets. The access token is loaded only for a valid renderable account.
It claims all eligible targets in that same transaction using:

```sql
UPDATE social_post_target
SET status = 'publishing', published_at = NULL, remote_post_id = NULL,
    error = NULL, updated_at = CURRENT_TIMESTAMP
WHERE uuid = $1 AND status IN ('draft', 'failed');
```

The transaction commits **before** any external HTTP request. Competing API
claims/edits serialize on the existing post lock; the conditional target update
is a second check. A post with any unresolved `publishing` target rejects the
whole next publish request with per-target conflicts, including targets that
have already failed while another target is still in progress. This prevents
an overlapping request from retrying an earlier target in an active batch.
The mechanism works across API processes, without an in-memory mutex, leases,
long-running transactions or database connections held during network calls.

Each result is persisted independently using a conditional update from
`publishing`. Success sets `published_at`, `remote_post_id`, clears `error`, and
updates `updated_at`. Definite failures set `failed`, clear `published_at` and
`remote_post_id`, store a safe error and update `updated_at`. `scheduled_at`
remains unchanged. A later failure cannot roll back earlier remote successes.

PUT and DELETE return 409 while the post has an unresolved claim. Existing CRUD
behavior for other statuses remains compatible, including deliberate deletion
of published targets/posts. Removing and recreating a target creates a new
identity and is outside the same-target duplicate guard.

## Mastodon and credentials

Accounts require `enabled`, `remote_account_id`, an HTTPS `base_url` origin and
a nonempty access token. Origin validation follows preview's rules (no userinfo,
query, fragment or path other than `/`) and additionally requires HTTPS.
There is no production HTTP exception; tests use local `httptest.NewTLSServer`
instances and explicitly injected trusted transports. Preview still accepts
HTTP metadata and does not require credentials.

The service's `SocialPublishingAccount` is separate from response metadata.
Its token is private and accidental JSON/formatting is redacted, following
Part 1's secret-input convention. Refresh tokens and expiry are not loaded;
there is no token refresh or separate `verify_credentials` call. Authorization
uses `Bearer` headers only, never URLs. Tokens are not logged, returned, placed
in Grains data, or included in errors. Continue to disable SQL parameter tracing
for the database pool as required in Part 1.

Text-only publishing sends form-encoded `status=<RenderedPost.Text>` and
`visibility=public` to `POST /api/v1/statuses`. With an image it first downloads
the selected rendered image and sends `POST /api/v2/media` with multipart `file`
and optional `description=ImageAlt`. The status form then includes `media_ids[]`.
Returned status/media IDs must be bounded decimal strings and cannot contain
the access token. Missing or malformed IDs never count as success.

Media uploads returning 202 or no processed URL are checked with
`GET /api/v1/media/:id`: at most five polls, a 500 ms delay and a 10-second total
polling deadline. HTTP 206 means still processing. No status is sent until
processing succeeds. This follows the [Mastodon media API](https://docs.joinmastodon.org/methods/media/).
Status form behavior follows the [status API](https://docs.joinmastodon.org/methods/statuses/).
The neighboring kulturbytes-social Mastodon CLI/README informed the upload/form,
alt-text, redirect, IP-binding and uncertain-outcome behavior. No Python, Click,
SQLite, journal or template structures were ported.

## Images and transport security

The pinned Pluto library exposes transformation through its HTTP handler, not
an exported internal function suitable for returning the same transformed
bytes. Reading original files directly would change the rendered image. The
publisher therefore uses the existing `ImageUrl()` API:

- HTTPS only, exact configured `BaseApiUrl` origin and prefix, canonical non-nil
  UUID path `/api/image/:uuid`; no URL credentials, fragments or encoded paths.
- Only Mastodon's generated `type=jpg`, `width=1920`, `height=1920` query
  parameters are accepted. Arbitrary external image URLs are rejected.
- The production transport resolves the host and rejects every non-public
  address, including loopback, link-local, RFC1918, IPv6 local/private,
  IPv4-mapped private addresses, multicast, reserved and transition ranges.
  It dials the validated numeric IP directly, retaining the original Host and
  verified TLS server name. A second DNS lookup cannot rebind the connection.
- Environment proxies are disabled and redirects are not followed for any
  download, upload, poll or status request. No Authorization is sent on images.
- Downloads require HTTP 200 and a declared JPEG, PNG or WebP content type,
  checked against the actual bytes. Content-Length and a bounded stream read
  enforce a 10 MiB limit. Remote JSON reads are limited to 1 MiB.
- Every HTTP request uses the caller's context. Image downloads have a
  15-second deadline, each remote API request 30 seconds, and each complete
  target operation 90 seconds. Client timeouts are capped at 30 seconds.
  Only bounded local result persistence (5 seconds) detaches cancellation so
  a known remote success can still be recorded after the caller disconnects.

Private-network Mastodon/Image API deployments are intentionally unsupported by
the default transport. Client injection is a trusted Go dependency for tests,
not a user-controlled request option or a configuration switch bypassing SSRF.

## Errors, crashes and limits before Part 6

Errors use fixed operation messages or HTTP status codes; raw remote bodies,
HTML pages, URLs, headers and transport/database errors are never echoed.
Examples are `Mastodon returned HTTP 401` or `image exceeds 10 MiB limit`.
No POST is automatically retried. Upload/poll failures can leave an unattached
remote media object; Part 5 does not attempt remote cleanup.

A status POST timeout, cancellation, connection failure, HTTP 5xx, invalid JSON
or missing/invalid status ID is **uncertain**: the remote server may have created
the status. The target stays `publishing` with a safe error and is blocked from
republishing. A definite status rejection such as 401/403/404/422/429 becomes
`failed` and is eligible for a later explicit manual request. This intentionally
prefers duplicate prevention when remote success cannot be ruled out.

A process can crash after remote success but before local persistence. A local
final-update failure has the same limitation. The durable `publishing` claim
then remains and must not be reset automatically, even if it is old. Stop any
active publishers and manually inspect the Mastodon account before resolving
the row: record an observed remote ID as `published`, or mark `cancelled` if the
outcome cannot safely be recovered. Only return it to `failed` after confirming
that no status was created. No recovery endpoint or automatic timeout reset is
provided. Claims created before a crash can also include targets not yet sent.

Part 5 does not provide exactly-once remote publication, durable content snapshots
or a publication journal. Source/account changes after preparation apply to later
requests; the in-flight request uses the prepared values. An explicit manual
retry renders current content again. Part 6/7 remain responsible for publication
history/revisions/hashes, idempotency, scheduling, workers, cron/background jobs,
automatic retries/queues, automation rules, webhooks, OAuth flows, token-refresh
daemons and dashboard UI. No event-creation hook publishes automatically.

## Verification and deliberate manual smoke test

Automated service tests use injected TLS mock servers/transports, never real
social services. API tests reuse the isolated-schema PostgreSQL/PostGIS harness,
including per-target persistence, partial success, permission checks, read-only
preview, matching rendered content, credentials/log secrecy, concurrent requests
from separate handlers, cancellation, claim/final-write failures and migrations.

```sh
export GOCACHE=/tmp/uranus-go-build
go test ./...
go vet ./...
go build ./...
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
URANUS_AUTH_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
  go test -race -count=1 ./...
```

Without explicit database environment variables, DB tests skip. A disposable
local PostgreSQL/PostGIS instance can supply them; never use the production DB.

A developer may deliberately run this real-world smoke test after deployment:

1. Apply the migration and select an existing event in an organization they edit.
2. Create an enabled Mastodon account through the existing account API with its
   actual account ID, public HTTPS instance origin and access token. Use secure
   credential input, without shell tracing or logged request bodies.
3. Create a social post referencing that event and the existing account target.
4. Call `/preview?lang=de`; inspect exact text, selected image and alt text.
5. Deliberately call `/publish?lang=de` once; expect 200 and a remote ID.
6. GET the post: check `published`, non-null `published_at` and `remote_post_id`,
   and null `error`. Inspect the public Mastodon post, image and alternative text.
7. Call `/publish` again: expect 409 and confirm that no second status exists.

Also test one text-only event and one image event against the intended instance.
Creating a real post is never part of automated verification.
