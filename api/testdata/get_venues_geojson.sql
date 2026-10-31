CREATE TABLE uranus.portal2 (
    uuid uuid PRIMARY KEY,
    geographic_filter text NOT NULL DEFAULT 'geometry',
    org_filter text NOT NULL DEFAULT 'all'
);
CREATE TABLE uranus.portal_geometry (
    uuid uuid PRIMARY KEY,
    portal_uuid uuid REFERENCES uranus.portal2(uuid),
    parent_uuid uuid REFERENCES uranus.portal_geometry(uuid),
    geometry geometry(MultiPolygon, 4326) NOT NULL
);
CREATE TABLE uranus.portal_org_allowlist (portal_uuid uuid, org_uuid uuid);
CREATE TABLE uranus.portal_org_blocklist (portal_uuid uuid, org_uuid uuid);
CREATE TABLE uranus.venue_type (key text PRIMARY KEY, marker_style jsonb);
CREATE TABLE uranus.pluto_image_link (
    context text, context_uuid uuid, identifier text, pluto_image_uuid uuid
);
CREATE TABLE uranus.venue (
    uuid uuid PRIMARY KEY,
    org_uuid uuid,
    type text,
    name text,
    street text,
    house_number text,
    city text,
    country text,
    scope text,
    point geometry(Point, 4326),
    building geometry,
    web_link text
);

INSERT INTO uranus.portal2 (uuid) VALUES
    ('00000000-0000-0000-0000-000000000001'),
    ('00000000-0000-0000-0000-000000000002');
INSERT INTO uranus.portal_geometry (uuid, portal_uuid, parent_uuid, geometry) VALUES
    ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', NULL,
     ST_Multi(ST_MakeEnvelope(0, 0, 10, 10, 4326))),
    ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001',
     '00000000-0000-0000-0000-000000000011', ST_Multi(ST_MakeEnvelope(0, 0, 5, 5, 4326)));
INSERT INTO uranus.portal_org_allowlist VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000021');
INSERT INTO uranus.portal_org_blocklist VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000022');

INSERT INTO uranus.venue (uuid, org_uuid, name, scope, point)
SELECT
    ('00000000-0000-0000-0000-' || lpad(id::text, 12, '0'))::uuid,
    ('00000000-0000-0000-0000-' || lpad(org_id::text, 12, '0'))::uuid,
    'Venue ' || id,
    scope,
    ST_SetSRID(ST_MakePoint(lon, lat), 4326)
FROM (VALUES
    (31, 21, 'shared', 2, 2),
    (32, 21, 'organization', 8, 8),
    (33, 22, 'shared', 3, 3),
    (34, 23, 'shared', 7, 7),
    (35, 21, 'shared', 12, 12),
    (37, 21, 'shared', 5, 5),
    (38, 23, 'shared', 10, 10)
) AS fixtures(id, org_id, scope, lon, lat);
