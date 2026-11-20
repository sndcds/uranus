# Venue-Scope

## Bestandsaufnahme vor der Änderung

Geprüfter Ausgangsstand: `ce426a0` (main). Die Suche nach `scope`, `Scope`,
`organization`, `shared` und `standard` umfasste Go, SQL, DDL, OpenAPI,
Dokumentation, Skripte, vorhandene Tests und die verfügbare Git-Historie.

| Funktion | Fundstellen und bisheriges Verhalten |
| --- | --- |
| Schreiben | `AdminCreateVenue`, POST `/api/admin/venue/create`: Inline-INSERT mit ausdrücklich übergebenem Scope; trimmt, prüft aber nur auf Leerstring. Einziger aktiver Venue-Insert im Repository. |
| Historischer Insert | `sql/admin-insert-venue.sql` ohne Scope; über `SqlInsertVenue` geladen und nur von `AdminUpsertVenue` referenziert. Dessen einzige Routenregistrierung ist auskommentiert. Der alte Pfad verwendet außerdem numerische `organization_id`/`RETURNING id`, während das aktuelle Schema UUIDs verwendet. Kein fachlicher Scope lässt sich daraus ableiten. |
| Updates | Aktiver `AdminUpdateVenueFields` und historisches `admin-update-venue.sql` enthalten kein Scope-Update. Scope ist über die aktuelle API nur bei Erstellung festlegbar. |
| Lesen/Durchreichen | `model.Venue`, `AdminGetVenue`/`admin-get-venue.sql`, `AdminGetOrgVenues`/`admin-get-org-venues.sql`, `AdminGetOrgChoosableVenues`/`admin-chooseable-venues.sql`, `GetChoosableVenues` sowie beide GeoJSON-SQL-Dateien geben den gespeicherten Scope weiter. |
| Filtern/Validieren | `GetVenuesGeoJSON` prüft die kommaseparierten `scopes` lokal gegen zwei Stringliterale; `get-venues-geojson.sql` und `get-portal-venues-geojson.sql` filtern über `v.scope = ANY(...)`. Die Datenbank-CHECK erlaubt dieselben beiden Werte. |
| Keine Scope-Ausgabe | `get-venues.sql`/`GetVenues` und `get-venue.sql`/`GetVenue` enthalten bisher keinen Scope. Deren API-Vertrag wird hier nicht erweitert. |
| Vertrag | Die beiden OpenAPI-Definitionen von `Venue.scope` in den Choosable-Venues-Verträgen dokumentieren bislang nur String/Beispiel, kein Enum. |

`eb1a936` (2026-08-06) führte den Scope in Erstellung, Modell und GeoJSON ein,
einschließlich der Werte `organization`/`shared`. Zuvor enthielt der aktive
Erstellungs-INSERT keinen Scope. `0ad340d` (2026-08-31) fügte erstmals
`ddl/venue.ddl` hinzu, bereits mit `DEFAULT 'standard'` und gleichzeitig nur
`organization`/`shared` in der CHECK-Constraint. Es gibt keinen früheren
Venue-DDL-Stand im verfügbaren Repository, der `standard` als gültigen Wert
belegt. Außer dem Default und dessen bisheriger Dokumentation wurde kein aktiver
Schreiber oder Scope-Filter für `standard` gefunden.

Mit aktivierter CHECK-Constraint scheitert ein INSERT ohne Scope am bisherigen
Default. Daraus folgt **keine Aussage über Produktionsdaten**: ältere oder
abweichende Live-Schemata und vor einer `NOT VALID`-Constraint vorhandene Daten
sind aus dem Repository nicht rekonstruierbar.

## Repository-Zielzustand

- `organization`: fachlich auf die jeweilige Organisation begrenzter Ort;
  Admin-Beschriftung „Provisorischer Ort (nicht eigener Ort)“.
- `shared`: eigenständiger / gemeinsam verwendbarer Venue-Datensatz;
  Admin-Beschriftung „Eigener Ort“.

`venue.scope` bleibt authoritative. Keine Ableitung aus `org_uuid`, Eigentum
oder Berechtigungen. `model/venue_scope.go` definiert den Stringtyp
`VenueScope`, die beiden Konstanten und `IsValid()`. Modelle und bestehende
Scope-Antwortfelder verwenden diesen Typ, ohne JSON-Feldnamen oder Werte zu
ändern. Die beiden OpenAPI-Venue-Schemas enthalten das entsprechende Enum.

`AdminCreateVenue` trimmt wie bisher und validiert vor Transaktion,
Berechtigungsabfrage und INSERT. `" organization "` wird deshalb weiterhin
als `organization` akzeptiert; `standard`, Leerstring, `foo`, `shared'`,
`Organization` und `SHARED` liefern HTTP 400. GeoJSON nutzt denselben Validator
für `scopes=organization`, `scopes=shared` und kommaseparierte Kombinationen,
einschließlich des bisherigen Trimmens einzelner Einträge. Fehlender/leerer
`scopes`-Parameter bedeutet weiterhin keine Scope-Einschränkung.

DDL: `scope text NOT NULL CHECK (scope IN ('organization', 'shared'))`, **ohne
Default**. Alle aktiven Inserts müssen Scope ausdrücklich setzen. Der historische
Upsert/SQL-Pfad bleibt unroutbar und ist als solcher kommentiert; seine
Reaktivierung erfordert eine separate Migration des UUID-/Request-Vertrags und
explizite Scope-Validierung. Ein AST-Test schützt vor stiller Routenaktivierung.

Über die vorhandene API kann Scope nach Erstellung weiterhin nicht geändert
werden. Dies ist kein Datenbank-Unveränderlichkeitsgebot: eine fachlich begründete,
gezielte Korrektur durch die Datenbankadministration bleibt möglich.

Der lesende Abgleich mit `sndcds/uranus-admin` bestätigt
`frontend/shared/contracts.ts` (`venueScopeSchema`), die Beschriftungen in
`frontend/app/utils/venues.ts` und den Filter in
`backend/app/repositories/entities.py`: das Admin-Backend filtert seine
`venue_scope`-Projektion aus `v.scope`. Dessen `/venues?scope=...` ist nicht
die öffentliche Uranus-`/api/venues`-Route. Hier werden weder neue Filter noch
Scope-Felder für bisher scope-freie Antworten hinzugefügt. Das Admin-Repository
wurde nicht geändert.

## Diagnose, Migration und Deployment

Das Repository verwaltet bereits versionierte SQL-Dateien in `migrations/`
(Refresh-Token-Migration), aber keinen automatischen Runner. Der vorhandene
Deploy-Workflow baut/startet die API; er führt keine Migrationen oder Tests aus.
Der Repository-Zielzustand beweist daher nicht den Live-Zustand.

1. Vor Deployment die read-only Prüfung mit einem geeigneten Datenbankzugang
   ausführen:

   ```sh
   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f sql_utils/diagnose-venue-scope.sql
   ```

   Sie zeigt Häufigkeiten, Anzahl unbekannter/NULL-Werte, Default, Nullable-Status
   sowie die tatsächlichen CHECK-Constraints und deren Validierungsstatus.
   Custom-Schemata konsistent in Diagnose/Migration anpassen.
2. Bei ungültigen Bestandswerten **anhalten**: Datensätze mit fachlich
   Verantwortlichen einzeln prüfen. Keine pauschale Zuordnung von `standard`,
   keine Ableitung aus Organisations-ID/Berechtigungen. Abweichende Constraints
   und externe Import-/Schreibprogramme ebenfalls vor dem Deployment prüfen;
   nicht im Repository enthaltene Schreiber sind durch diese Prüfung nicht belegt.
3. Wenn keine ungültigen Werte vorliegen, Migration vor dem API-Rollout ausführen
   und Version `202609250002` im Deployment protokollieren:

   ```sh
   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/202609250002_venue_scope.up.sql
   ```

   Sie sperrt die Tabelle, prüft die Daten erneut und bricht bei ungültigen
   Werten atomar ab. Erst danach entfernt sie den Default, erzwingt NOT NULL
   und ersetzt die bekannte `venue_scope_check` durch die geprüfte Zwei-Werte-
   Constraint. Andere Constraints bleiben erhalten. Keine Zeile wird geändert
   oder gelöscht. Die Sperre verhindert Rennen mit Schreibern; bei fünf Sekunden
   Lock-Wartezeit bricht die Migration ab und kann später wiederholt werden.
   Für große Tabellen ein Wartungsfenster planen, da die Prüfung Zeilen liest.
4. Die korrigierte API deployen. Der zuvor aktive Erstellungs-Handler übergibt
   Scope bereits explizit und benötigt keinen Default; die neue Version weist
   ungültige Eingaben zusätzlich vor dem DB-Zugriff zurück.
5. Diagnose erneut ausführen: keine ungültigen Werte, kein Default, NOT NULL
   und validierte Constraint. Erstellung beider Typen und Scope-Filter prüfen.

Die Korrektur ist vorwärtsgerichtet. Die `.down.sql` bricht ausdrücklich ab,
statt den ungültigen Default wiederherzustellen oder Validierung zu lockern.
Ein Rollback der Anwendung kann das korrigierte Schema behalten.
Es wurde keine Produktionsdatenbank abgefragt oder migriert.

## Prüfungen

`go test ./...` enthält gezielte Tests für gültige/ungültige Scope-Werte,
JSON-Ausgabe, Validierung vor DB-Zugriff, GeoJSON-Fehler, exakte DDL-/OpenAPI-
Wertemengen und die deaktivierte Legacy-Route.

Fokussierte DB-Tests mit expliziter Wegwerf-Datenbank (PostGIS verfügbar):

```sh
URANUS_VENUE_TEST_DATABASE_URL="$TEST_DATABASE_URL" GOMAXPROCS=2 \
  go test -p 2 ./api -run VenueScopePostgres -count=1
```

Die Tests verwenden isolierte Schemas, echte Venue-DDL/SQL-Dateien und kleine,
leere abhängige Tabellen für die Abfrageregression. Sie prüfen Erstellung beider
Scopes ohne Default, Whitespace-Kompatibilität, Scope-Ausgabe aller bereits
scope-führenden Endpunkte, GeoJSON-Filter mit/ohne Portal, Migration des alten
Defaults sowie unveränderte Daten/Schema bei fehlgeschlagener Vorprüfung.
