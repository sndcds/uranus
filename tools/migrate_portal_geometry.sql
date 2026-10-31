-- Manuelle Migration fuer portal2; die Anwendung gleichzeitig umstellen.
-- Aufrufe von portal2.geometry und geometry_mode muessen angepasst werden.
-- Anleitung: docs/portal-geometry.md
-- Ausfuehren: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f tools/migrate_portal_geometry.sql
-- Pro vorhandener Geometrie entsteht ein eigener Datensatz. Gleiche Flaechen
-- werden nicht automatisch zusammengefuehrt. Portale ohne Geometrie behalten NULL.

BEGIN;

-- Verhindert Aenderungen zwischen Kopieren und Entfernen der alten Spalten.
LOCK TABLE uranus.portal2 IN ACCESS EXCLUSIVE MODE;

-- SRID 4326 entspricht den vorhandenen venue.point-Daten. Insbesondere SRID 0
-- darf nicht ohne Kenntnis des tatsaechlichen Koordinatensystems ersetzt werden.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM uranus.portal2
        WHERE geometry IS NOT NULL
          AND (
              ST_GeometryType(geometry) NOT IN ('ST_Polygon', 'ST_MultiPolygon')
              OR ST_SRID(geometry) <> 4326
              OR ST_NDims(geometry) <> 2
              OR NOT ST_IsValid(geometry)
          )
    ) THEN
        RAISE EXCEPTION 'Migration abgebrochen: Es werden gueltige 2D-Polygone/MultiPolygone mit SRID 4326 erwartet.';
    END IF;
END;
$$;

CREATE TABLE uranus.portal_geometry (
    uuid uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by uuid REFERENCES uranus."user"(uuid) ON DELETE SET NULL,
    created_at timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_by uuid REFERENCES uranus."user"(uuid) ON DELETE SET NULL,
    modified_at timestamp without time zone,
    name text NOT NULL CHECK (btrim(name) <> ''),
    description text NOT NULL CHECK (btrim(description) <> ''),
    geometry geometry(MultiPolygon, 4326) NOT NULL,
    CONSTRAINT portal_geometry_valid CHECK (ST_IsValid(geometry))
);

CREATE INDEX portal_geometry_geometry_gist_idx
    ON uranus.portal_geometry USING gist (geometry);

-- Erst zuordnen und kopieren, danach den Fremdschluessel hinzufuegen.
ALTER TABLE uranus.portal2 ADD COLUMN geometry_uuid uuid;

UPDATE uranus.portal2
SET geometry_uuid = gen_random_uuid()
WHERE geometry IS NOT NULL;

INSERT INTO uranus.portal_geometry (
    uuid, created_by, created_at, modified_by, modified_at,
    name, description, geometry
)
SELECT
    p.geometry_uuid, p.created_by, p.created_at, p.modified_by, p.modified_at,
    'Geometrie: ' || p.name,
    COALESCE(
        NULLIF(btrim(p.description), ''),
        'Geometrie des Portals "' || p.name || '".'
    ),
    ST_Multi(p.geometry)
FROM uranus.portal2 p
WHERE p.geometry IS NOT NULL;

ALTER TABLE uranus.portal2
    ADD CONSTRAINT portal2_geometry_uuid_fkey
    FOREIGN KEY (geometry_uuid) REFERENCES uranus.portal_geometry(uuid)
    ON DELETE RESTRICT;

CREATE INDEX portal2_geometry_uuid_idx ON uranus.portal2 (geometry_uuid);

-- Vor dem Entfernen pruefen, dass jede vorhandene Geometrie uebernommen wurde.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM uranus.portal2 p
        LEFT JOIN uranus.portal_geometry g ON g.uuid = p.geometry_uuid
        WHERE p.geometry IS NOT NULL
          AND ST_AsEWKB(g.geometry) IS DISTINCT FROM ST_AsEWKB(ST_Multi(p.geometry))
    ) THEN
        RAISE EXCEPTION 'Migration abgebrochen: Die Geometrien wurden nicht vollstaendig uebernommen.';
    END IF;
END;
$$;

-- Der bisherige GiST-Index auf portal2.geometry wird dabei ebenfalls entfernt.
--ALTER TABLE uranus.portal2
--    DROP COLUMN geometry,
--    DROP COLUMN geometry_mode;

COMMIT;

-- Beispiel fuer den anschliessenden Zugriff:
-- SELECT p.uuid, p.name, p.geometry_uuid,
--        g.name AS geometry_name, g.description AS geometry_description,
--        ST_AsGeoJSON(g.geometry)::jsonb AS geometry
-- FROM uranus.portal2 p
-- LEFT JOIN uranus.portal_geometry g ON g.uuid = p.geometry_uuid;
