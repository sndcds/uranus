# Social scheduling and worker (Part 7)

Builds on [social accounts](social-accounts.md), [posts and targets](social-posts.md),
[ContentItems](content-items.md), [rendering and preview](social-preview.md),
[manual publishing](social-publishing.md), and [publication history and recovery](social-publications.md).

Schedule decides when. The renderer decides what. The publisher decides how.
Publication history records what happened.

## Manual publishing also uses the worker

`POST /api/admin/social/posts/:uuid/publish[?lang=de|da|en]` is now asynchronous.
After authorization it queues eligible targets with `status = scheduled`,
`scheduled_at = CURRENT_TIMESTAMP` and `publication_source = manual`, and returns
`202 Accepted`. Already queued targets are conflicts. At least one accepted target
produces 202 with individual acceptance/conflict results; no eligible targets
produces 409. No source rendering, credential read, fingerprint or publication
attempt occurs in the API process.

This deliberately replaces the Part-5/6 synchronous 200/207 contract. Clients read
the existing post and publication-history APIs for results. Cancelling the caller's
HTTP context after commit does not cancel the request. The optional explicit label
language is stored as `publish_language`; it is a rendering preference, not a content
snapshot. Content is always read at execution time.

Manual means immediately eligible, not immediate remote delivery. Processing starts
on the next available poll and is subject to earlier due work and the batch limit.
The same worker process handles manual and future requests. It has a separate heap
and process lifetime from the event API; systemd can enforce separate resource budgets.
Preview remains a synchronous API operation.

## Scheduling API

`POST /api/admin/social/posts/:uuid/schedule`

```json
{"scheduled_at":"2026-09-27T18:00:00+02:00"}
```

Supply a future RFC3339 timestamp with `Z` or an explicit offset. Missing, null,
malformed, offset-free and past timestamps return 400. Validation uses Go's
standard time decoder and checks the database clock again after acquiring the
post lock. The database stores `timestamptz`; the worker compares against
`CURRENT_TIMESTAMP`. Unknown body fields are rejected, including target IDs.

The action sets every eligible target of the post to `scheduled`, replaces
`scheduled_at`, sets `publication_source = scheduled`, clears `publish_language`
and its current error, and updates `updated_at`. There is no
target-selection parameter. Existing remote IDs and publication timestamps remain
available until a later result replaces them.

`POST /api/admin/social/posts/:uuid/cancel`

No body is required. This cancels all scheduled targets of the post, including
manual requests awaiting execution. It retains
`scheduled_at` for auditing and updates `updated_at`. Other targets are unchanged.
A committed publishing claim cannot be cancelled, including during a remote call.

Both routes use the existing admin JWT and cookie-origin middleware. Accepted
organization membership and `UserPermEditOrg` are checked inside the transaction.
Authorization belongs to the scheduling action; execution needs no user JWT or
user permission check. A user's later departure does not remove the organization's
schedule. The worker is an internal process with direct database access.

Successful responses use HTTP 200 and the existing Grains envelope, with
`response_type` equal to `admin-schedule-social-post` or `admin-cancel-social-post`.
`data` is the complete updated `SocialPost`, exactly as in post GET/PUT responses:
its `uuid`, organization/source/audit fields, and `targets` array including each
target's `uuid`, `status`, `scheduled_at`, `published_at`, and other current fields.

| Condition | HTTP |
|---|---|
| Missing/invalid authentication | 401 |
| Missing membership or organization permission | 403 |
| Invalid post UUID or schedule payload/time | 400 |
| Post not found | 404 |
| No eligible targets, including a post with no targets | 409 |
| Any unresolved `publishing` target in the post | 409 |

Actions are atomic across the post. A publishing conflict rejects the entire
action; it never leaves a partially rescheduled or cancelled set.

## State transitions

| Current target state | Explicit schedule | Cancel | Worker |
|---|---|---|---|
| `draft` | `scheduled` | unchanged | ignored |
| `failed` | `scheduled` | unchanged | ignored |
| `scheduled` | replace time | `cancelled` | claim when due |
| `published` | `scheduled` | unchanged | ignored |
| `cancelled` | unchanged | unchanged | ignored |
| `publishing` | post-wide 409 | post-wide 409 | ignored |

If all targets are unchanged/ineligible, the action returns 409. Cancelled targets
are final in Part 7, including for manual publishing. Post CRUD retains this state;
there is no reset endpoint. Explicit scheduling from `published` is allowed so that
changed content can be published later without rendering at scheduling time.
Scheduling from `failed` is an explicit operator action, never an automatic retry.

Manual publishing accepts `draft`, `failed` and `published` targets for queuing.
It does not consume an existing schedule or accept cancelled targets. Other eligible
targets may be accepted alongside individual conflicts. A `published` legacy target
with neither success metadata nor a published history row cannot be queued or
scheduled: replacing its status would otherwise lose the only duplicate guard.
An unresolved publishing target blocks the whole post for manual queuing,
schedule/cancel and further worker claims.

`scheduled_at` survives claim, success, failure, uncertainty and cancellation.
Success sets `published_at` and `remote_post_id`. A reschedule replaces the target's
current schedule time; this field is not a complete audit log of schedule edits.

## Execution and shared publishing path

The existing root binary has a dedicated process mode for every publication:

```sh
go build -o uranus .
./uranus --config config.json --social-worker
./uranus --config config.json --social-worker --once
```

The entry point calls `ApiHandler.RunSocialWorker(ctx, once)`. No API router,
Pluto server, interactive prompt, TTY, daemon fork, PID file or health server is
started. It uses the existing configuration/database/SQL initialization; therefore
run from the repository deployment directory containing `sql/`. Existing shared
configuration validation, including a nonempty `jwt_secret`, still applies,
although the worker neither obtains nor sends admin JWTs. It never prints the
configuration or credentials.

Optional JSON configuration, with defaults:

```json
{
  "social_worker_interval": 30,
  "social_worker_batch_size": 20
}
```

Interval is seconds, accepted range 15–3600. Batch size is 1–100. Invalid values
fail startup. The worker polls immediately, processes at most one batch, then
waits the configured interval after that batch. An empty queue also waits and
produces no error. `--once` processes at most one batch and exits; repeat it for
additional batches. `--once` without `--social-worker` is an error.

Manual and scheduled targets follow one internal execution path:

1. Lock the post and select one current due target.
2. Load current source data and account metadata, render and fingerprint.
3. Apply Part-6 idempotency. For a real attempt, insert `social_publication` and
   set the target to `publishing` in the same transaction; commit.
4. Call the shared publisher, outside all database transactions and row locks.
5. Finalize publication and target together through the existing Part-6 helper.

No API request preclaims publications, and no batch of future remote calls is preclaimed. Processing is sequential within
one process, and the batch limit bounds attempts or duplicate completions per poll.

**Content is rendered at execution time, not schedule time.** Scheduling creates
no snapshot and reads no credentials. An event scheduled at 10:00, edited at 15:00
and processed at 18:00 publishes the current 15:00 data. Current account credentials
and metadata are loaded during the claim. The exact rendered payload and its
fingerprint are committed in publication history before the remote mutation.

## Concurrency and due query

Due targets satisfy `status = 'scheduled' AND scheduled_at <= CURRENT_TIMESTAMP`.
Candidate order is `scheduled_at, created_at, uuid`. The query joins the owning post
and uses `LIMIT 1 FOR UPDATE OF p SKIP LOCKED`, preserving the existing post-first
lock order used by manual publishing and CRUD. A second query after acquiring the
post lock rechecks due state and unresolved siblings, and locks the target.

Multiple workers skip locked posts and work on other posts. After commit, the
`publishing` state prevents another claim. Thus two workers on one due target
produce one publication attempt and one remote status POST. A remote failure or
uncertain outcome is not picked up on subsequent polls. The existing partial
unique publication indexes also protect unresolved and successful attempts.

Cancel/reschedule use the same post lock. If cancel wins, there is no publish. If
reschedule wins with a future time, there is no early publish. If the worker claims
first, both actions return 409 while it is publishing. A schedule arriving after
successful finalization can explicitly schedule the now-published target again;
idempotency still applies at execution. No operation resets an active claim.

Order is deterministic for available work in one worker. Concurrent processes,
locked posts and remote latency can change global completion order. A post with
an unresolved publishing sibling waits for Part-6 reconciliation; other posts
continue normally.

## Publication history and idempotency

`social_post_target.publication_source` records the pending request origin; its
optional `publish_language` preserves an explicit manual label-language choice.
The worker copies the request origin to the immutable
`social_publication.publication_source`. Existing
attempts default to `manual` because no worker existed before Part 7. This field is
included in list/detail/reconcile responses and protected by the existing immutable
snapshot trigger. No `scheduled_by`, credential snapshot or separate job table is
introduced. Targets remain queue state; publications remain attempt history.

The Part-6 fingerprint and per-target publishing context are unchanged. Both
manual and timed requests perform duplicate checks in the worker.
When an identical successful fingerprint already exists, the worker makes no remote
call and creates no duplicate publication. It sets the scheduled target to
`published`, restores the matching publication's remote ID and finish time, clears
its current error, and retains `scheduled_at`. This also handles content reverting
to an older successful snapshot. The recorded publication time is the original
success time, not the duplicate-check time.

A changed fingerprint creates a new attempt. Part-5 legacy successes with no
snapshot remain conservatively protected; a current render cannot reconstruct a
historical fingerprint. Existing remote ID/time are retained in that case. Legacy published targets
without any success metadata/history remain unqueueable, including through schedule.

## Failures, shutdown and recovery

| Outcome | Target | Publication | Next poll |
|---|---|---|---|
| Successful remote publish | `published` | `published` | ignored |
| Missing/invalid source, render failure, disabled account, missing token | `failed` | `failed` | ignored |
| Unimplemented platform, definite remote rejection | `failed` | `failed` | ignored |
| Uncertain remote result, timeout, 5xx or invalid success response | `publishing` | `uncertain` | ignored |
| Remote result cannot be finalized in DB | `publishing` | committed `publishing` | ignored |
| Database claim/read/commit failure | unchanged | no committed new attempt | next normal poll |

Local preflight failures create failed history even when no valid render snapshot
exists. Missing source records are sanitized local errors. Database failures abort
the claim and never authorize a remote call. Definite/uncertain classification,
including image-upload behavior, follows the existing publisher exactly. For
example, uncertainty about a media upload does not imply a public status exists;
see [manual publishing](social-publishing.md).

Target failures do not stop processing other eligible posts. Claim/database errors
end the current batch; continuous mode logs a sanitized error and waits for the next
normal poll. One-shot exits 1 on such an error. Recorded target failures, uncertain
outcomes and finalization failures are logged and remain visible in history; they
do not make one-shot exit nonzero. Operators must inspect history for outcomes.
Initialization/configuration errors exit nonzero.

SIGTERM/SIGINT cancel the worker context and running remote requests. The existing
bounded, five-second local finalizer can still record the observed result after
cancellation. An interrupted remote status request remains uncertain and requires
[Part-6 reconciliation](social-publications.md). A process crash after claim remains
recoverable in the same way. No worker automatically repairs or retries it.

Deletion follows existing foreign keys: organization/post deletion cascades through
scheduled targets without history. Protected publication history and unresolved
claims can prevent deletion, as established in Parts 5/6; no dangling queue is added.

Logs go to stderr using the existing standard logger. They include startup/shutdown,
claimed target/publication UUIDs and outcomes, never tokens, authorization headers,
rendered text or raw remote bodies.

## Migration and query index

Apply `migrations/202609260002_social_scheduling.up.sql` after the Part-6 migration,
using the repository's existing manual SQL deployment process. It adds request origin and optional label language to targets, immutable
publication origin to history, and a partial ordered index:

```sql
CREATE INDEX social_post_target_due_idx
    ON uranus.social_post_target (scheduled_at, created_at, uuid)
    WHERE status = 'scheduled';
```

An `EXPLAIN (ANALYZE, BUFFERS)` experiment with 50,000 synthetic targets, including
1,000 scheduled targets, showed the original separate indexes sorting all due rows
and scanning the post table before `LIMIT 1`. The partial index allowed ordered
access and a primary-key lookup of the first available post without that sort.
This is a synthetic plan check, not a production throughput benchmark.

Canonical DDL files include both additions. Stop workers/API publishers before
rollback; cancel pending schedules and reconcile unresolved attempts first. The
down migration refuses scheduled/publishing targets. Dropping the source columns
loses manual/scheduled attribution; export history before intentional rollback.

## systemd example

The existing deployment workflow builds `/opt/dev/go/uranus/uranus` as `oklab`.
A separate unit can run that same binary:

```ini
[Unit]
Description=Uranus Social Publishing Worker
Wants=network-online.target
After=network-online.target postgresql.service

[Service]
Type=simple
User=oklab
WorkingDirectory=/opt/dev/go/uranus
ExecStart=/opt/dev/go/uranus/uranus --config /opt/dev/go/uranus/config.json --social-worker
Restart=on-failure
RestartSec=10
TimeoutStopSec=15
# Example resource budget; size for the deployment and representative images.
CPUWeight=20
CPUQuota=100%
MemoryHigh=256M
MemoryMax=512M
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

The sample gives the worker less CPU weight than default sibling services and
caps it to one CPU's worth of time. `MemoryHigh` applies memory pressure before the
hard `MemoryMax` boundary, which can kill the worker. These are illustrative budgets,
not measured sizing recommendations. Size them for actual images and the host;
a worker killed after claim still needs normal Part-6 reconciliation. See the
[upstream systemd resource-control reference](https://github.com/systemd/systemd/blob/main/man/systemd.resource-control.xml).
The shared database and host I/O remain common resources.

Install as `uranus-social-worker.service` after adapting configuration ownership
and database location to the deployment. This PR does not install or enable it.
The existing deployment workflow restarts only `uranus.service`; restart the worker
unit separately after deployment. Use `journalctl -u uranus-social-worker.service`
and publication history for operations. Continuous mode can stay alive during a
database outage, so process status alone does not prove that jobs are progressing.

## Verification and scope

Tests extend the existing `URANUS_SOCIAL_TEST_DATABASE_URL` harness, use disposable
PostgreSQL/PostGIS schemas and local `httptest` servers, and never contact social
platforms. Coverage includes API transitions/authorization, due/future/cancelled
work, deterministic order and batch limits, two workers, cancel/reschedule races,
current source/account data, idempotency, failure history and recovery, shutdown,
one-shot, migration rollback and organization cascades.

```sh
export GOCACHE=/tmp/uranus-go-build
go test ./...
go vet ./...
go build ./...
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
URANUS_AUTH_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
  go test -race -count=1 ./...
```

Automatic retries are intentionally not implemented. There is no retry backoff,
retry counter, dead-letter queue, automation rule, event-triggered scheduling,
schedule-edit history, dashboard UI, extra metrics library or scheduler framework.
PostgreSQL remains the source of truth. Facebook, Instagram and Bluesky publishing
remain unimplemented. Cancellation is terminal in this part; individual-target
selection is not supported. Timed rendering follows the source language at
execution; manual requests may store an explicit label-language preference.
