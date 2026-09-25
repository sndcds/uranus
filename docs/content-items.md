# Content items (Part 3)

`model.ContentItem` is the runtime representation of an Uranus source for the
social-content process. It is neither a social post nor rendered output.
There is no content table, new source identity, migration, or HTTP endpoint.
The model contains no database connection or platform-specific behavior.

## Loading

Internal admin code uses the existing `ApiHandler` and authenticated Gin context:

```go
item, txErr := h.LoadContentItem(gc, orgUuid, "event", eventUuid, "de")
```

For a persisted post, use its authoritative organization and source reference:

```go
item, txErr := h.LoadSocialPostContentItem(gc, postUuid, "de")
```

Both calls use `WithTransaction`, the existing pool/schema, `ApiTxError`, and
`CheckAllOrgPermissionsTx` with `UserPermEditOrg` and accepted organization
membership. The post variant uses the existing post permission/row-lock helper.
Source rows are locked for sharing through completion to stabilize ownership
and source fields during loading. No partial item is returned on any query or
transaction failure. Database details are neither logged nor retained in
returned internal errors.

Source UUIDs and organization UUIDs must be valid non-nil UUIDs. Only `event`,
`venue`, and `organization` are accepted. Missing authentication returns an
internal 401 error, invalid input 400, insufficient organization permissions
403, and a source missing from the expected organization 404. The latter also
covers an existing foreign source, without exposing it. An organization source
must itself be the expected organization; events and venues must have matching
`org_uuid`. A related event venue can belong to another organization, as allowed
by the existing event/venue model. This does not make that venue eligible as a
standalone source for the event organization's posts.

## Source mapping

Common fields identify the original source (`source_type`, `source_uuid`) and
its organization (`org_uuid`). No UUID is generated for a ContentItem.

| ContentItem field | Event | Venue | Organization |
| --- | --- | --- | --- |
| `title` | `event.title` | `venue.name` | `organization.name` |
| `subtitle` | `event.subtitle` | absent | absent |
| `description` | stored description | stored description | stored description |
| `summary` | `event.summary` | `venue.summary` | absent |
| `content_language` | `content_iso_639_1` | `content_iso_639_1` | `content_iso_639_1` |
| `url` | `event.source_link` | `venue.web_link` | `organization.web_link` |
| `ticket_link` | stored ticket link | stored ticket link | absent |
| `online_link` | stored online link | absent | absent |
| `location` | base event venue | venue address | organization address |
| `dates` | all stored event dates | empty | empty |
| `images` | event image links | venue image links | organization image links |

The loader reads the same base tables used by the detail handlers. Event search
projections contain denormalized source data, not translated content; loading
content does not require generating or refreshing them. A later call reflects
current source values rather than an independently persisted snapshot.

Text is copied verbatim, including any stored Markdown or HTML. No text is
rendered, stripped, summarized, translated, or replaced with invented defaults.
Missing optional data stays absent and collections are non-nil empty arrays.
Only stored source/web/ticket/online URLs are used; no canonical frontend URL is
constructed when a source link is missing.

`ContentLocation` contains the existing UUID, name, address components and web
link. `ContentDate` contains the original event-date UUID, local date/time
values, duration, all-day flag, ticket link, effective location and space
UUID/name. Dates are ordered by start date, start time and UUID. Venue/space
inheritance follows `get-event-dates.sql`: a date venue overrides the event
venue, and its explicit space is used; otherwise the event venue and space are
used. Date-specific ticket links remain separate from the base event ticket
link, as in the date projection. All occurrences remain available without selecting one, filtering
by publishing state, inventing an end time, or attaching a timezone. Date and
time strings use the detail API's `YYYY-MM-DD` and `HH24:MI` conventions.

## Language and images

`language` is the requested label language normalized by `app.NormalizeLocale`
using the configured supported languages and its existing German fallback.
`content_language` separately records the stored source-text language. Uranus
currently has one title/description per source; changing `language` does not
translate those fields.

Images reuse `model.Image` and `ImageUrl`, loading existing
`pluto_image_link`/`pluto_image` records in identifier/UUID order. All linked
identifiers are preserved; no main-image choice or logo inheritance is made.
Original dimensions, focus values, alternative text, description, creator,
copyright, license and AI label are copied. License labels use `license_i18n`
for the requested language and the existing `all-rights-reserved` fallback from
the public detail queries. Missing label rows are not translated automatically.
No images are downloaded, generated, resized or cropped.

## Existing concepts and compatibility

The repository previously had no `ContentItem` or general `Content` domain
model. `ShareMeta` in `api/internal_tests.go` is an existing OpenGraph-specific
view and remains separate. The neighboring Python `kulturbytes-social` project
has its own source-neutral `ContentItem`; this change introduces no Python
dependency, renderer adapter, publication identity or mapping framework.

Part 2 continues storing `source_type` and `source_uuid` without a polymorphic
foreign key. Its CRUD requests/responses and all target metadata remain
unchanged. A post can still be saved before its source exists; loading its
content then returns 404. No eager content loading is added to post CRUD.

A pre-existing regression in the post-create handler logged raw database
errors. Three error-print statements were removed to restore the existing
credential-safety tests; request/response behavior is unchanged.

## Verification and remaining scope

Unit tests cover invalid UUIDs/types, authentication and sanitized errors.
PostgreSQL tests use the existing isolated-schema Social test harness and actual
source DDL. They cover each source type, missing data/sources, localization,
image metadata, date/venue/space inheritance, organization isolation and
permissions, post integration without mutations, fresh source values and failed
relation queries without partial results. The fixture declares the enums
referenced by the source DDL because no enum creation scripts are checked in.

```sh
go test ./...
go vet ./...
URANUS_SOCIAL_TEST_DATABASE_URL="$TEST_DATABASE_URL" \
URANUS_AUTH_TEST_DATABASE_URL="$TEST_DATABASE_URL" go test -race -count=1 ./...
```

Rendering, templates, preview, OpenGraph generation, image processing,
publishing, platform APIs, scheduling/workers, retries, publication history,
idempotency and UI remain outside Part 3. No renderer interface is introduced.
