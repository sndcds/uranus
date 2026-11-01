# Portal-Geometrien auslagern

## Öffentlicher Endpunkt für die Auswahlhierarchie

`GET /api/portal/:uuid/geometries` liefert die Geometriehierarchie eines Portals
ohne Authentifizierung. `:uuid` ist die UUID aus `portal2`.
Die Antwort verwendet den üblichen `apiRequest`-Rahmen mit `service`,
`api_version`, `response_type: "get-portal-geometries"`, `status`, `timestamp`
und `metadata` (einschließlich `portal_uuid` und `response_time_ms`).
Der `data`-Bereich sieht beispielsweise so aus:

```json
{
  "geometry": {
    "uuid": "73a57380-a4c0-4eb9-a4e3-5b4428b85dc9",
    "name": "Stadt",
    "description": "Gesamtes Stadtgebiet",
    "order": 0,
    "admin_level": 8,
    "type": "city",
    "child_types": [
      { "type": "city-district", "count": 1 }
    ],
    "childs": [
      {
        "uuid": "09202887-6f7e-4da8-8471-7549cd151b51",
        "name": "Stadtteil",
        "description": "Gebiet des Stadtteils",
        "order": 0,
        "admin_level": 10,
        "type": "city-district",
        "child_types": [],
        "childs": []
      }
    ]
  }
}
```

Die Zuordnung erfolgt bei allen Knoten über `portal_geometry.portal_uuid`.
Die Hauptgeometrie hat `parent_uuid IS NULL`; `portal2.geometry_uuid` wird für
diesen Endpunkt nicht verwendet. Kinder werden über `parent_uuid` zugeordnet
und können beliebig viele weitere Ebenen enthalten. Geschwister sind nach
`order`, dann `name`, dann `uuid` aufsteigend sortiert. Ein SQL-`NULL` in
`order` wird als `0` behandelt; fehlende Werte für `admin_level` und `type`
bleiben JSON-`null`. `childs` ist immer ein Array, auch bei Blattknoten.
`child_types` zählt pro Knoten die direkten Kinder desselben Portals nach
`type`, alphabetisch nach Typ sortiert. Kinder mit `type: null` werden dabei
nicht gezählt. Gibt es keine Kinder mit gesetztem Typ, ist `child_types` ein
leeres Array (`[]`). Enkel und weitere Nachfahren zählen jeweils nur bei
ihrem direkten Elternknoten.
Polygonkoordinaten sind nicht Teil dieser Antwort; sie können über den
bestehenden Endpunkt `/api/portal/:uuid/geojson?geometry=<geometry_uuid>`
abgerufen werden.

Ein vorhandenes Portal ohne Geometrien liefert HTTP 200 mit
`data: {"geometry": null}`. Eine ungültige UUID liefert HTTP 400, ein
unbekanntes Portal HTTP 404. Fehlerhafte Hierarchien (mehrere oder keine
Hauptgeometrie trotz vorhandener Knoten, portalübergreifende Eltern oder
Zyklen) sowie Datenbankfehler liefern HTTP 500.

Der PostgreSQL-Integrationstest verwendet ein eigenes temporäres Schema:

```sh
URANUS_PORTAL_TEST_DATABASE_URL="postgres://..." go test ./api -run TestGetPortalGeometries -v
```

## Manuelle Migration

Das manuell auszuführende Skript
[`tools/migrate_portal_geometry.sql`](../tools/migrate_portal_geometry.sql)
verschiebt die Geometrien aus `uranus.portal2` nach `uranus.portal_geometry`.
Die Verknüpfung erfolgt über `portal2.geometry_uuid → portal_geometry.uuid`.

## Datenmodell

Eine Geometrie erhält eine eigene UUID, einen Namen, eine Beschreibung und die
bisherigen Erstellungs- und Änderungsinformationen des Portals. Name und
Beschreibung sind Pflichtfelder und dürfen nicht leer sein. Gespeichert werden
gültige zweidimensionale `MultiPolygon`-Geometrien mit SRID 4326; einzelne
Polygone werden mit `ST_Multi` übernommen.

Pro Portal mit einer vorhandenen Geometrie entsteht zunächst ein eigener
Geometriedatensatz. Gleiche Flächen werden nicht automatisch zusammengeführt.
Der Name lautet zunächst `Geometrie: <Portalname>`. Eine vorhandene
Portalbeschreibung wird übernommen; bei fehlender oder leerer Beschreibung
wird `Geometrie des Portals "<Portalname>".` eingesetzt.

`geometry_uuid` bleibt für Portale ohne Geometrie `NULL`. Mehrere Portale können
später auf dieselbe Geometrie verweisen. `ON DELETE RESTRICT` verhindert das
Löschen einer noch verwendeten Geometrie. Die Allowlist und Blocklist sowie
`filter_type` werden durch die Migration nicht geändert.

## Anwendung vorbereiten

Die Migration entfernt `portal2.geometry` und `portal2.geometry_mode`.
Die folgenden bestehenden Zugriffe müssen beim Einspielen auf das neue Schema
umgestellt werden:

- `sql/get-portal2.sql`: Geometrie per Join lesen, Portalspalten mit `p.`
  qualifizieren und `geometry_mode` aus der Ergebnismenge entfernen.
- `api/get_portal.go`: In `GetPortal2` das Feld `GeometryMode` und den passenden
  `Scan`-Parameter entfernen; dynamische Bedingungen auf `p.uuid` und `p.slug`
  qualifizieren.
- `sql/get-portal-geojson.sql` und `sql/get-portal-venues-geojson.sql`: Die neue
  Geometrietabelle einbinden und die räumlichen Prüfungen darauf umstellen.
- `api/get_events.go` und `sql/portal-condition.sql`: Den Portal-Join um die
  Geometrietabelle ergänzen und die bisherigen Zugriffe auf `p.geometry`
  ersetzen. Für Portale ohne Geometrie weiterhin einen `LEFT JOIN` verwenden.

Diese Anpassungen am Anwendungscode sind nicht Bestandteil des manuellen
SQL-Skripts. Sie müssen vor dem Wiederanlauf der API bereitstehen.

## Migration ausführen

1. Eine Datenbanksicherung erstellen und den Bestand prüfen:

   ```sql
   SELECT
       ST_GeometryType(geometry) AS geometry_type,
       ST_SRID(geometry) AS srid,
       ST_NDims(geometry) AS dimensions,
       ST_IsValid(geometry) AS valid,
       count(*)
   FROM uranus.portal2
   WHERE geometry IS NOT NULL
   GROUP BY 1, 2, 3, 4;
   ```

   Erwartet werden gültige 2D-Polygone oder Multipolygone mit SRID 4326.
   Abweichende Werte müssen vorab anhand ihres tatsächlichen
   Koordinatensystems beziehungsweise ihrer Geometrie korrigiert werden.
   Das Skript weist andere SRIDs einschließlich SRID 0 zurück.

2. Die Portal-API für das Einspielen anhalten. Das Skript sperrt `portal2`
   während der Transaktion exklusiv; auch lesende Zugriffe müssen warten.

3. Die Migration einmalig aus dem Repository-Verzeichnis ausführen:

   ```sh
   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f tools/migrate_portal_geometry.sql
   ```

   Das Skript legt die Tabelle und Indizes an, kopiert die Daten und setzt den
   Fremdschlüssel. Vor dem Entfernen der alten Spalten vergleicht es die
   übernommenen Geometrien mit dem Bestand. Jeder Fehler führt zum Rollback
   der gesamten Migration. Nach einem erfolgreichen Commit erfordert die
   Rückkehr zum bisherigen Schema eine gesonderte Rückmigration oder die
   Wiederherstellung der Sicherung.

4. Die Zuordnung prüfen und die angepasste API starten:

   ```sql
   SELECT
       p.uuid,
       p.name,
       p.geometry_uuid,
       g.name AS geometry_name,
       g.description AS geometry_description,
       ST_AsGeoJSON(g.geometry)::jsonb AS geometry
   FROM uranus.portal2 p
   LEFT JOIN uranus.portal_geometry g ON g.uuid = p.geometry_uuid;
   ```

Die Geometrietabelle besitzt einen räumlichen GiST-Index. Der Fremdschlüssel
`portal2.geometry_uuid` erhält einen separaten Index; der bisherige Index auf
`portal2.geometry` entfällt beim Entfernen der Spalte.
