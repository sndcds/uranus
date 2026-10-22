# Manual social publishing (Part 5)

[Part 7: scheduling and worker](social-scheduling.md) adds planned execution using
the shared publishing and publication-history pipeline.

Part 5 builds on [social accounts](social-accounts.md), [posts and targets](social-posts.md),
[ContentItem loading](content-items.md) and [rendering/preview](social-preview.md).

[Part 6](social-publications.md) extends this workflow with durable snapshots,
fingerprints, publication history and manual reconciliation.

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
Mastodon's response URL is not returned or stored. Part 6 adds publication
history while retaining the current target's `remote_post_id`.

## Migration and endpoint

Apply the Part-5, Part-6 and [Part-7 migrations](social-scheduling.md#migration-and-query-index)
in order. Part 7 changes manual publishing to asynchronous execution in a separate
worker process. The API never renders, reads credentials or contacts the platform
in response to a publish request. Deploy the worker alongside the updated API.

`POST /api/admin/social/posts/:uuid/publish[?lang=de|da|en]`

No body is required. Existing JWT/cookie-origin protection and accepted membership
with `UserPermEditOrg` apply. The handler locks the post and queues its eligible
stored targets in one transaction. A queued target has `status = scheduled`,
`scheduled_at = CURRENT_TIMESTAMP` and `publication_source = manual`. The optional
normalized language preference is stored as `publish_language`; without it the
worker follows the current source language. No content snapshot is created here.

HTTP 202 means accepted for processing, not published. The Grains envelope retains
`response_type: admin-publish-social-post` and its existing post/results structure:

```json
{
  "post_uuid": "01994126-6680-7000-8000-000000000030",
  "results": [
    {
      "target_uuid": "01994126-6680-7000-8000-000000000031",
      "social_account_uuid": "01994126-6680-7000-8000-000000000020",
      "platform": "mastodon",
      "status": "scheduled",
      "publication_source": "manual"
    }
  ]
}
```

When some targets are ineligible, their results retain their current status and
contain an error; the request still returns 202 if at least one target was queued.
An unresolved `publishing` target blocks the entire post. Queued targets have no
new publication UUID, fingerprint or remote result yet. Poll the existing post
GET and publication-list endpoints to observe execution; failures are persisted
there, rather than returned later through the original HTTP request.

| HTTP status | Meaning |
| --- | --- |
| 202 | At least one target queued; inspect individual acceptance results |
| 409 | No target eligible, or unresolved publishing work in the post |
| 400 | Invalid UUID or no targets |
| 401 / 403 | Missing authentication / membership / permission or cookie-origin rejection |
| 404 | Missing post |
| 500 | Enqueue transaction failed; no new work committed |

Missing sources, invalid accounts, rendering errors and remote failures are
execution outcomes in publication history. `200` and `207` synchronous publishing
responses are replaced by `202`; clients must migrate accordingly. An identical
successful fingerprint is also checked later by the worker, without a new remote
post or duplicate attempt. Legacy published targets without a snapshot or any
success metadata remain conservatively rejected instead of losing their guard.

## State and concurrency

```text
draft / failed / published
          │ explicit manual POST
          ▼
scheduled (manual, available now)
          │ worker claim and snapshot, COMMIT
          ▼
publishing ── success ──────────── published
          ├── definite failure ── failed
          └── uncertain ───────── publishing (reconciliation required)
```

An already queued/scheduled target cannot be queued again. Cancel can remove a
pending manual request before the worker claims it; an explicit schedule can move
it into the future and replaces its source with `scheduled`. Cancelled targets are
terminal in this part. No automatic retry or `/retry` endpoint exists.

The [worker](social-scheduling.md) handles both manual and timed requests through
the same render/fingerprint/claim/publish/finalize path. It loads current content,
account metadata and credentials at execution time. A claim takes one target at a
time under the existing post lock. Its publication snapshot and `publishing` state
commit before any external call; no database lock or transaction spans remote HTTP.
A post with unresolved publishing work waits for reconciliation, while other posts
continue. Multiple processes coordinate entirely through PostgreSQL.

Success stores the remote ID and publication time; failures retain any earlier
success metadata. The request time in `scheduled_at` remains available for auditing.
A duplicate completion restores the matching successful publication's ID/time,
including an older snapshot if source content reverted. PUT/DELETE cannot remove
unresolved claims or protected publication history.

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

## Errors, crashes and recovery

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
final-update failure has the same limitation. Part 6 leaves a durable `publishing`
publication and target, recoverable via the local [reconciliation API](social-publications.md#uncertainty-and-manual-recovery).
Stop all active publishers and inspect the remote account before confirming an
abandoned `publishing` attempt. Completed uncertain outcomes are recorded as
`social_publication.status = uncertain`. Neither state permits an automatic retry.

Part 6 supplies durable content snapshots, fingerprints and attempt history;
source/account changes after preparation still apply only to later requests.
Part 7 adds the shared manual/scheduled worker. Exactly-once remote publication,
automatic retries, automation rules, webhooks, OAuth/token-refresh daemons and
dashboard UI remain outside this workflow.

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
5. Deliberately call `/publish?lang=de` once; expect 202. Run the worker or wait
   for its next poll, then inspect publication history.
6. GET the post: check `published`, non-null `published_at` and `remote_post_id`,
   and null `error`. Inspect the public Mastodon post, image and alternative text.
7. Call `/publish?lang=de` again after completion: expect 202. After the worker
   processes it, confirm no second remote status or duplicate attempt exists.

Also test one text-only event and one image event against the intended instance.
Creating a real post is never part of automated verification.
