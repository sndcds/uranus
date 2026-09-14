# Authentication

The existing HS256 signing key, `app.Claims`, Gin context keys (`user-uuid`,
`jwt-claims`), user model and response envelopes remain in use. `app.ParseJWT`
validates HS256, the signature, required expiration and JWT `nbf`. It does not
select a token purpose. API middleware and the optional access-token UUID helper
accept only `token_type=access` and a nonzero UUID. Activation also uses the common
parser and continues comparing the entire token against the stored activation
token. Invitations retain their separate claims and HS256 parser; neither
activation nor invitation tokens authenticate normal API requests.

## Endpoints and compatibility

- `POST /api/login`: validates the existing credentials and active user, signs
  access and refresh JWTs and registers the refresh token in a transaction before
  sending cookies or the existing JSON response.
- `POST /api/admin/refresh`: authenticates the `refresh_token` cookie directly,
  outside `JWTMiddleware`. When no refresh cookie is present, the existing Bearer
  transport is supported. Checks type, UUID, `jti`, active user, database ownership,
  expiration and revocation. Returns the previous JSON fields and Authorization
  response header, plus `refresh_token` and both updated cookies.
- `POST /api/admin/logout`: uses the same refresh-token authentication, revokes the
  session's whole family and deletes both cookies. Repeated logout with a valid,
  already revoked token succeeds. Missing/invalid tokens return 401 and clear
  cookies; database failures return 500 without claiming a successful logout.

The requested `/auth/refresh` spelling was interpreted as illustrative: the
existing `/api/admin/refresh` URL is preserved. Existing Bearer login and refresh
payloads remain compatible: the dashboard already replaces a returned refresh
token. Full dashboard integration still requires server logout and cross-tab
refresh coordination as described below. Browser clients opting into cookies must
use `credentials: include`
for login, refresh, logout and authenticated calls. The existing dashboard clears
its local token state on logout; clients must call the new server logout endpoint
before clearing that state to obtain server-side revocation.

New access tokens expire after `min(auth_token_expiration_time, 900)` seconds;
the default is 900 seconds. Refresh lifetime retains
`refresh_token_expiration_time` (default seven days). Nonpositive access lifetimes
or refresh lifetimes no longer than access lifetimes fail issuance. Existing
access tokens signed with the configured key retain their old expiry; legacy refresh tokens without a
registered `jti` are rejected, so existing users will need to log in again.

The existing initializer decoded `Config.JwtSecret` but never assigned `JwtKey`.
Login and access validation consequently used an empty HMAC key. Initialization
now requires a nonempty jwt_secret and assigns that existing secret to JwtKey
before connecting to the database. Parsing and session issuance also fail closed
without a key. Previously empty-key-signed access tokens are intentionally invalid;
this is a necessary security correction. Config.Print redacts the JWT secret.

## Persistence, rotation and concurrent use

Only refresh-token metadata is persisted, never JWT strings or access tokens.
The UUIDv7 generator already used by the project creates each new `jti`. The first
`jti` also becomes a database-only `family_uuid`, inherited by successors.

Login, refresh and logout use the existing `WithTransaction` / `ApiTxError` / pgx
infrastructure and `h.DbSchema`. A `FOR UPDATE` lock on the user row serializes
session mutations for that user, including different generations of a family.
This intentionally favors simple correctness over parallel refreshes for the same
user. Rotation revokes the old token and inserts its replacement in one
transaction; cookies are emitted only after successful commit. Expiration is
checked against the live database clock after acquiring the lock, not just the
transaction's start time.

Reusing a revoked, still cryptographically valid refresh token commits revocation
of all active members of that family, then returns the same generic 401 used for
other invalid refresh tokens. Other sessions for the user remain active. Two
simultaneous refreshes of the same token produce at most one successful response;
the second revokes that response's successor as reuse. Clients must serialize
refreshes (including across browser tabs) and must not retry an already consumed
token after losing a response. The dashboard's existing promise prevents duplicate
refreshes within one tab only.

Logout also revokes descendants when it receives a recently rotated token. Access
JWTs remain valid until expiration. There is no separate session-management API;
operators can revoke all active rows for a user or family through existing
administrative database procedures, taking the same user-row lock in that
transaction. Account deactivation prevents refresh. Password reset behavior is
otherwise unchanged. Families have a sliding refresh lifetime; there is no new
absolute session timeout or automatic cleanup job. Expired rows can be removed by
maintenance because expired JWTs cannot be presented for refresh or reuse checks.

## Cookies and CSRF

Previously login and refresh returned JSON tokens only; middleware already had an
`access_token` cookie fallback. This change adds host-only cookies (no Domain),
HttpOnly, Secure in production (`dev_mode=false`), SameSite=Lax, matching JWT
Expires/MaxAge. Access uses Path=/api; refresh uses Path=/api/admin, covering both
session endpoints. Deletion uses the exact same attributes and paths.

The checked deployment uses api.kulturbytes.de and app.kulturbytes.de, which are
same-site. The pre-existing CORS allowlist is retained: https://app.kulturbytes.de
and http://localhost:5173. Login now uses that credentialed allowlist instead of
public wildcard CORS. Arbitrary configured `allow_origins: ["*"]` is not used for
authentication. Localhost calling the production API is cross-site, so SameSite=Lax
cookies are not a suitable transport there; existing Bearer behavior remains
available. Local HTTP cookie development requires explicit dev_mode=true.

Cookie-authenticated mutations, refresh and logout require a trusted Origin or,
when Origin is absent, a trusted Referer origin. Missing or untrusted provenance
returns 403 without mutating session state or cookies. Explicit Bearer access
requests do not need this cookie CSRF check. Refresh cookies take precedence over
Bearer headers, so adding a header cannot bypass the cookie check. Browser login
also validates provenance; nonbrowser JSON login without browser headers remains
supported. SameSite alone would not protect against an untrusted sibling domain.
New frontend origins must be added to the shared `app.AllowedAuthOrigin` allowlist.

## Migration and validation

The repository previously contained DDL snapshots under `ddl/`, with duplicated
indexes in some snapshots, but no migration runner or version history. The first
versioned forward/rollback migration is therefore stored in `migrations/`:

- `202609140001_refresh_token.up.sql`: transactional creation of
  `uranus.refresh_token`, foreign key with cascading user deletion, primary key,
  lifetime constraint and user/family/expiry indexes.
- `202609140001_refresh_token.down.sql`: transactional removal of that table.

Apply the forward file once through your deployment's SQL migration step before
starting the updated API, recording version 202609140001 there. For a controlled
test/staging migration, `psql "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f
migrations/202609140001_refresh_token.up.sql` executes the transaction. No automatic
startup migration or production database change is performed. The existing SSH
deployment has no migration step; rollout must arrange this before restarting the
service. Rollback destroys registered sessions and must only be used deliberately.
The checked schema is `uranus`; deployments using a different schema must adapt
the migration consistently with `db_schema` before applying it.

Run unit/static checks with `go test ./...` and `go vet ./...`. PostgreSQL tests are
opt-in with `URANUS_AUTH_TEST_DATABASE_URL` pointing at a disposable test database:

```sh
URANUS_AUTH_TEST_DATABASE_URL="$TEST_DATABASE_URL" go test -race ./...
```

Each integration test creates and removes its own unique schema, takes the actual
user DDL table definition, and runs the real refresh migration. Tests cover login,
rotation, ownership, inactive/deleted users, unknown/revoked/expired tokens, logout,
family isolation, simultaneous requests, insert/commit rollback, cookie behavior,
foreign keys, uniqueness and migration up/down/up. Without the explicit variable,
integration tests report a skip; no application database configuration is read.

## Validation of this change

Validated on 2026-09-14 with Go 1.26.0 and an isolated PostgreSQL 18.6 instance:

- `go test ./...`: unit tests passed.
- `URANUS_AUTH_TEST_DATABASE_URL=... go test -race -count=1 ./...`: all packages passed, including real PostgreSQL integration and migration tests.
- `go vet ./...` and `git diff --check`: passed.

No production database, local ignored configuration, frontend code, or dependency
versions were changed.

## Changed files

- `api/admin_login.go`
- `api/admin_signup.go`
- `api/api_utils.go`
- `api/auth_integration_test.go`
- `api/auth_tokens.go`
- `api/auth_tokens_test.go`
- `app/auth_origin.go`
- `app/config.go`
- `app/jwt.go`
- `app/jwt_test.go`
- `app/middleware.go`
- `app/uranus.go`
- `app/uranus_test.go`
- `docs/authentication.md`
- `migrations/202609140001_refresh_token.down.sql`
- `migrations/202609140001_refresh_token.up.sql`
- `uranus-api.go`
- `uranus-api_test.go`
