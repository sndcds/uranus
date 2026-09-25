# Social rendering and preview (Part 4)

## Architecture and migration from kulturbytes-social

`ContentItem → service.SocialRenderer → model.RenderedPost`

`NewSocialRenderer(platform, frontendClient)` selects Facebook, Instagram,
Mastodon or Bluesky. Rendering is a pure transformation with no database,
credentials, HTTP client, clock, publication state or publishing methods.
`model.SocialTargetPreview` adds the target UUID only at the API boundary.

The existing Python implementation was inspected before implementation:
`common/src/kulturbytes_common/rendering.py`, `text_limits.py`, `image_urls.py`,
`data/templates/kulturbytes/*.j2`, and `data/sources/kulturbytes.yaml`.
Its canonical content/rendered-output boundary, summary-before-description
selection, platform layouts, Kulturbytes/city hashtags and Pluto transformations
are carried into Uranus. Bluesky is supported in both projects.

Uranus takes precedence where the implementations differ:

- The existing Go ContentItem, models, service layer and Markdown/GFM parser are
  reused. No Python process, Jinja runtime, configurable source adapter or second
  content model is introduced. Platform layout logic is implemented in Go.
- Every supplied occurrence is included in loader order, including its inherited
  or overridden venue. There is no date selection based on the current time.
- Oversized posts fail explicitly. The Python renderer's automatic shortening
  and selective metadata removal are intentionally not ported.
- All platforms receive plain text; Markdown/HTML formatting is removed, visible
  text and HTTP(S) link destinations are retained. Raw markup is never executed.
- Fields unavailable in ContentItem, such as tags, prices and organizer names,
  are not invented. Instagram uses the existing single-image approach.

## API

`POST /api/admin/social/posts/:uuid/preview[?lang=de|da|en]`

No request body is required. Authentication uses the existing admin JWT
middleware, including cookie-origin protection. The caller needs
`UserPermEditOrg` and accepted membership in the post's organization.

The handler loads the persisted post and targets, resolves the current source
using Part 3, reads only the account metadata needed for rendering, and renders
one preview per target. A repeatable-read transaction keeps the source relations
and account configuration in one database snapshot. Existing permission/source
row locks are reused. Targets retain their existing created-at/UUID ordering.

Accounts must exist, be enabled, belong to the post organization and have a
remote account ID. Mastodon requires `base_url`; any supplied base URL must be
an HTTP(S) origin without credentials, path, query or fragment. Credentials and
token expiry are not loaded or validated, so tokenless previews work. Existing
publication status does not prevent preview.

Success is HTTP 200 in the existing Grains envelope. Example (illustrative IDs
and envelope timestamp; absent optional URLs are omitted):

```json
{
  "service": "API",
  "api_version": "1.0",
  "response_type": "admin-preview-social-post",
  "status": 200,
  "timestamp": "2026-09-25T12:00:00Z",
  "metadata": {"response_time_ms": 2},
  "data": {
    "post_uuid": "01994126-6680-7000-8000-000000000030",
    "previews": [
      {
        "target_uuid": "01994126-6680-7000-8000-000000000031",
        "platform": "instagram",
        "text": "📅 Jazzabend\n🗓 01.10.2026 · 19:30 Uhr\n\n👉 https://events.example.test/de/veranstaltung/01994126-6680-7000-8000-000000000010/202610011930\n#Kulturbytes",
        "image_url": "https://api.example.test/api/image/01994126-6680-7000-8000-000000000016?ratio=4%3A5&type=jpg&width=1080",
        "url": "https://events.example.test/de/veranstaltung/01994126-6680-7000-8000-000000000010/202610011930"
      }
    ]
  }
}
```

The rendered `data` is deterministic for identical content, targets and config;
the normal Grains timestamp and response-time metadata remain request-specific.

| Status | Meaning |
| --- | --- |
| 400 | Invalid UUID, no targets, invalid/disabled account or rendering failure |
| 401 | Missing or invalid authentication |
| 403 | Missing organization permission/membership or rejected cookie origin |
| 404 | Post absent, or source absent from the post's organization |
| 500 | Sanitized database/transaction failure |

Rendering fails atomically: no partial previews are returned if any target
fails. The existing envelope's `message` identifies the target UUID, platform
and cause, e.g. `target … (platform instagram): an image is required` or
`target … (platform mastodon): text exceeds limit of 500 characters (got 612);
content was not truncated`. A missing shared source is reported before rendering.

## Text, language and public URLs

Summary is preferred when nonblank, otherwise description. The source text is
never translated. Uranus currently has one title/description per source, not
per-language translations. Without `lang`, preview labels follow
`ContentLanguage`; an explicit `lang` selects the label language. Both use
`app.NormalizeLocale` and its German fallback. German, Danish and English labels
are provided, and the existing numeric date style is retained.

Public links use existing `frontend-client` configuration and the client routes:

| Source | German | Danish | English |
| --- | --- | --- | --- |
| Event | `/de/veranstaltung/:uuid/:date` | `/da/begivenhed/:uuid/:date` | `/en/event/:uuid/:date` |
| Venue | `/de/ort/:uuid` | `/da/sted/:uuid` | `/en/venue/:uuid` |

The first occurrence supplies the main event link. Its identifier uses the
existing `YYYYMMDDHHmm` slug. When the time is missing, the supported date UUID
route is used instead of inventing midnight. All occurrences remain in the text.
Without a usable frontend route/configuration, the existing ContentItem URL
(source/web link) is retained. There is no public organization detail route in
the client, so organizations use their existing web link. Missing URLs are
omitted; malformed supplied URLs produce a rendering error. Production domains
are not hardcoded.

## Offline platform policies

| Platform | Text budget | Image transformation |
| --- | --- | --- |
| Facebook | 63,206 Unicode code points (local preview policy) | 1200:630, width 1920, WebP |
| Instagram | 2,200 Unicode code points | Required; 4:5, width 1080, JPEG |
| Mastodon | 500 graphemes; HTTP(S) URLs reserve 23 each | Free ratio, JPEG; longest edge 1920 when dimensions are known |
| Bluesky | 300 graphemes and 3,000 UTF-8 bytes | 4:3, width 1920, JPEG |

Mastodon uses the [documented defaults](https://docs.joinmastodon.org/user/posting/),
without instance discovery; actual instance settings can differ. Bluesky follows
the [post lexicon](https://github.com/bluesky-social/atproto/blob/main/lexicons/app/bsky/feed/post.json).
The existing Instagram caption policy is retained. These checks are offline
rendering checks, not confirmation that a platform will accept a publication.

Pluto URLs come from the existing ContentItem image infrastructure. Selection
prefers `main` for events, `main_photo` for venues and `main_logo` for
organizations, falling back to the first image in the deterministic loader
order. Transformation parameters replace conflicting crop/size/format values,
while preserving unrelated query parameters. Only one limiting edge is set.
No image is downloaded, converted locally, stored or cached by preview. Byte
size, availability and delivered MIME type are not verified remotely.

## Scope and verification

No schema changes, publication calls, credential updates, post/target writes,
status changes, history, workers, scheduling, retries, automation or UI are added.
No publisher is implemented.

`service/social_render_test.go` covers all four platforms, localization, normal
and oversized content, exact boundaries, Unicode graphemes/bytes, URL accounting,
Pluto transformation/selection, missing images, account validation, source URLs,
all occurrences, summary fallback, plain text and deterministic nonmutation.
`api/social_preview_test.go` covers authentication/CSRF, UUID validation, the
Grains envelope, every platform, source language and explicit fallback, all
source types, permissions, missing/foreign sources, target errors and sanitized
failures. Database write-rejection triggers protect posts, targets and accounts
on successful and failing previews; stored credentials and publication metadata
are compared. HTTP transport is blocked during the side-effect test.

Run against an explicitly supplied disposable PostgreSQL database with PostGIS:

```sh
export GOCACHE=/tmp/uranus-go-build
go test ./...
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
URANUS_AUTH_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
  go test -race -count=1 ./...
go vet ./...
go build ./...
```

Without the test database environment variables, PostgreSQL tests are skipped.
The only added dependency is `github.com/rivo/uniseg` for Unicode grapheme counting.
