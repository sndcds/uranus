WITH
    params AS (
        SELECT $1::uuid AS org_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Organization
 * ------------------------------------------------------------------
 */
    org AS (
        SELECT o.*
        FROM {{schema}}.organization o
        JOIN params p ON p.org_uuid = o.uuid
    ),

/*
 * ------------------------------------------------------------------
 * Venues belonging to the organization
 * ------------------------------------------------------------------
 */
    venues AS (
        SELECT v.*
        FROM {{schema}}.venue v
        JOIN params p ON p.org_uuid = v.org_uuid
    ),

    venue_ids AS (
        SELECT uuid
        FROM venues
    ),

/*
 * ------------------------------------------------------------------
 * Spaces belonging to the organization's venues
 * ------------------------------------------------------------------
 */
    spaces AS (
        SELECT s.*
        FROM {{schema}}.space s
        JOIN venue_ids v ON v.uuid = s.venue_uuid
    ),

    space_ids AS (
        SELECT uuid
        FROM spaces
    ),

/*
 * ------------------------------------------------------------------
 * Events belonging to the organization
 *
 * Event has its own org_uuid, so this is the authoritative
 * organization relationship.
 * ------------------------------------------------------------------
 */
    events AS (
        SELECT e.*
        FROM {{schema}}.event e
        JOIN params p ON p.org_uuid = e.org_uuid
    ),

    event_ids AS (
        SELECT uuid
        FROM events
    ),

/*
 * ------------------------------------------------------------------
 * Event dates
 * ------------------------------------------------------------------
 */
    event_dates AS (
        SELECT ed.*
        FROM {{schema}}.event_date ed
        JOIN event_ids e ON e.uuid = ed.event_uuid
    ),

    event_date_ids AS (
        SELECT uuid
        FROM event_dates
    ),

/*
 * ------------------------------------------------------------------
 * Organization members
 * ------------------------------------------------------------------
 */
    organization_members AS (
        SELECT oml.*
        FROM {{schema}}.organization_member_link oml
        JOIN params p ON p.org_uuid = oml.org_uuid
    ),

/*
 * Users explicitly associated with the organization.
 *
 * This deliberately does NOT include users merely because they
 * created/modified an organization object.
 * ------------------------------------------------------------------
 */
    organization_users AS (
        SELECT DISTINCT u.*
        FROM {{schema}}."user" u
        WHERE u.uuid IN (
            SELECT oml.user_uuid
            FROM organization_members oml

            UNION

            SELECT uol.user_uuid
            FROM {{schema}}.user_organization_link uol
            JOIN params p ON p.org_uuid = uol.org_uuid
        )
    ),

/*
 * ------------------------------------------------------------------
 * Organization access grants
 * ------------------------------------------------------------------
 */
    organization_access_grants AS (
        SELECT oag.*
        FROM {{schema}}.organization_access_grants oag
        JOIN params p
            ON p.org_uuid = oag.src_org_uuid
                OR p.org_uuid = oag.dst_org_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Organization partner requests
 *
 * Note: the DDL currently does not declare FKs here.
 * ------------------------------------------------------------------
 */
    organization_partner_requests AS (
        SELECT opr.*
        FROM {{schema}}.organization_partner_request opr
        JOIN params p
            ON p.org_uuid = opr.from_org_uuid
                OR p.org_uuid = opr.to_org_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Event-related tables
 * ------------------------------------------------------------------
 */
    event_links AS (
        SELECT el.*
        FROM {{schema}}.event_link el
        JOIN event_ids e ON e.uuid = el.event_uuid
    ),

    event_type_links AS (
        SELECT etl.*
        FROM {{schema}}.event_type_link etl
        JOIN event_ids e ON e.uuid = etl.event_uuid
    ),

    user_event_links AS (
        SELECT uel.*
        FROM {{schema}}.user_event_link uel
        JOIN event_ids e ON e.uuid = uel.event_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Projection tables
 *
 * event_projection has an explicit org_uuid.
 * event_date_projection is connected through event_date.
 * ------------------------------------------------------------------
 */
    event_projections AS (
        SELECT ep.*
        FROM {{schema}}.event_projection ep
        JOIN params p ON p.org_uuid = ep.org_uuid
    ),

    event_date_projections AS (
        SELECT edp.*
        FROM {{schema}}.event_date_projection edp
        JOIN event_ids e ON e.uuid = edp.event_uuid
    ),

/*
 * ------------------------------------------------------------------
 * User ↔ organization
 * ------------------------------------------------------------------
 */
    user_organization_links AS (
        SELECT uol.*
        FROM {{schema}}.user_organization_link uol
        JOIN params p ON p.org_uuid = uol.org_uuid
    ),

/*
 * ------------------------------------------------------------------
 * User ↔ venue
 * ------------------------------------------------------------------
 */
    user_venue_links AS (
        SELECT uvl.*
        FROM {{schema}}.user_venue_link uvl
        JOIN venue_ids v ON v.uuid = uvl.venue_uuid
    ),

/*
 * ------------------------------------------------------------------
 * User ↔ space
 * ------------------------------------------------------------------
 */
    user_space_links AS (
        SELECT usl.*
        FROM {{schema}}.user_space_link usl
        JOIN space_ids s ON s.uuid = usl.space_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Favorite lists belonging to the organization
 * ------------------------------------------------------------------
 */
    favorite_lists AS (
        SELECT fl.*
        FROM {{schema}}.favorite_list fl
        JOIN params p ON p.org_uuid = fl.org_uuid
    ),

    favorite_list_ids AS (
        SELECT uuid
        FROM favorite_lists
    ),

    favorites AS (
        SELECT f.*
        FROM {{schema}}.favorite f
        JOIN favorite_list_ids fl ON fl.uuid = f.list_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Portals
 * ------------------------------------------------------------------
 */
    portal2 AS (
        SELECT p2.*
        FROM {{schema}}.portal2 p2
        JOIN params p ON p.org_uuid = p2.org_uuid
    ),

    portal_temp AS (
        SELECT pt.*
        FROM {{schema}}.portal_temp pt
        JOIN params p ON p.org_uuid = pt.org_uuid
    ),

    portal_org_allowlist AS (
        SELECT poa.*
        FROM {{schema}}.portal_org_allowlist poa
        JOIN params p ON p.org_uuid = poa.org_uuid
    ),

    portal_org_blocklist AS (
        SELECT pob.*
        FROM {{schema}}.portal_org_blocklist pob
        JOIN params p ON p.org_uuid = pob.org_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Display presets
 * ------------------------------------------------------------------
 */
    display_presets AS (
        SELECT dp.*
        FROM {{schema}}.display_preset dp
        JOIN params p ON p.org_uuid = dp.org_uuid
    ),

/*
 * ------------------------------------------------------------------
 * Pluto images
 *
 * pluto_image_link is polymorphic:
 *
 *   context      = 'organization'
 *   context_uuid = organization.uuid
 *
 *   context      = 'event'
 *   context_uuid = event.uuid
 *
 *   context      = 'venue'
 *   context_uuid = venue.uuid
 *
 *   context      = 'space'
 *   context_uuid = space.uuid
 *
 * ------------------------------------------------------------------
 */
    pluto_image_links AS (
        SELECT pil.*
        FROM {{schema}}.pluto_image_link pil
        WHERE
            (pil.context = 'organization'
                AND pil.context_uuid IN (SELECT uuid FROM org))

           OR (pil.context = 'event'
            AND pil.context_uuid IN (SELECT uuid FROM events))

           OR (pil.context = 'venue'
            AND pil.context_uuid IN (SELECT uuid FROM venues))

           OR (pil.context = 'space'
            AND pil.context_uuid IN (SELECT uuid FROM spaces))
    ),

    pluto_image_ids AS (
        SELECT DISTINCT pluto_image_uuid
        FROM pluto_image_links
        WHERE pluto_image_uuid IS NOT NULL
    ),

    pluto_images AS (
        SELECT pi.*
        FROM {{schema}}.pluto_image pi
                 JOIN pluto_image_ids i
                      ON i.pluto_image_uuid = pi.uuid
    ),

    pluto_cache AS (
        SELECT pc.*
        FROM {{schema}}.pluto_cache pc
                 JOIN pluto_image_ids i
                      ON i.pluto_image_uuid = pc.pluto_image_uuid
    )

/*
 * ==================================================================
 * Return all records
 * ==================================================================
 *
 * table_name tells us where each JSON record came from.
 */
SELECT 'organization' AS table_name, to_jsonb(x) AS record
FROM org x

UNION ALL
SELECT 'venue', to_jsonb(x)
FROM venues x

UNION ALL
SELECT 'space', to_jsonb(x)
FROM spaces x

UNION ALL
SELECT 'event', to_jsonb(x)
FROM events x

UNION ALL
SELECT 'event_date', to_jsonb(x)
FROM event_dates x

UNION ALL
SELECT 'event_date_projection', to_jsonb(x)
FROM event_date_projections x

UNION ALL
SELECT 'event_projection', to_jsonb(x)
FROM event_projections x

UNION ALL
SELECT 'event_link', to_jsonb(x)
FROM event_links x

UNION ALL
SELECT 'event_type_link', to_jsonb(x)
FROM event_type_links x

UNION ALL
SELECT 'user_event_link', to_jsonb(x)
FROM user_event_links x

UNION ALL
SELECT 'organization_member_link', to_jsonb(x)
FROM organization_members x

UNION ALL
SELECT 'user_organization_link', to_jsonb(x)
FROM user_organization_links x

UNION ALL
SELECT 'organization_access_grants', to_jsonb(x)
FROM organization_access_grants x

UNION ALL
SELECT 'organization_partner_request', to_jsonb(x)
FROM organization_partner_requests x

UNION ALL
SELECT 'user', to_jsonb(x)
FROM organization_users x

UNION ALL
SELECT 'user_venue_link', to_jsonb(x)
FROM user_venue_links x

UNION ALL
SELECT 'user_space_link', to_jsonb(x)
FROM user_space_links x

UNION ALL
SELECT 'favorite_list', to_jsonb(x)
FROM favorite_lists x

UNION ALL
SELECT 'favorite', to_jsonb(x)
FROM favorites x

UNION ALL
SELECT 'portal2', to_jsonb(x)
FROM portal2 x

UNION ALL
SELECT 'portal_temp', to_jsonb(x)
FROM portal_temp x

UNION ALL
SELECT 'portal_org_allowlist', to_jsonb(x)
FROM portal_org_allowlist x

UNION ALL
SELECT 'portal_org_blocklist', to_jsonb(x)
FROM portal_org_blocklist x

UNION ALL
SELECT 'display_preset', to_jsonb(x)
FROM display_presets x

UNION ALL
SELECT 'pluto_image_link', to_jsonb(x)
FROM pluto_image_links x

UNION ALL
SELECT 'pluto_image', to_jsonb(x)
FROM pluto_images x

UNION ALL
SELECT 'pluto_cache', to_jsonb(x)
FROM pluto_cache x

ORDER BY table_name;