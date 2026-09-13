# Empfehlung für ein Administrations-Dashboard

Stand: 13. September 2026. Untersucht wurde der lokale Stand `b257ae9` des Repositories `sndcds/uranus`, insbesondere `ddl/` sowie zugehörige SQL-Abfragen und Go-Handler. Das Verzeichnis heißt `ddl`, nicht `dll`.

Die Aufgabe ist gut machbar. Die vorhandenen Tabellen erlauben bereits eine Übersicht neuer Datensätze und viele Qualitätsprüfungen. Empfohlen wird eine Erweiterung der bestehenden Administration um eine priorisierte Arbeitsliste, einen Aktivitätsbereich und Datenqualitätsprüfungen.

Dies ist eine Schema- und Codeanalyse, keine Prüfung produktiver Daten. Die aufgeführten Risiken sind im beschriebenen Schema möglich; sie sind keine nachgewiesenen fehlerhaften Datensätze. DDL-Dateien sind Postico-Exporte, ausdrücklich kein vollständiges Datenbank-Backup. Vor einer Implementierung müssen Constraints, Funktionen, Enum-Werte und Zeitzonen der tatsächlich eingesetzten Datenbank abgeglichen werden.

## 1. Die tägliche Startseite

Die Startseite sollte direkt beantworten: Was ist neu? Was muss ich heute bearbeiten? Was beeinträchtigt bereits veröffentlichte Inhalte?

| Bereich | Anzeige | Aktion |
| --- | --- | --- |
| Neu eingegangen | Anzahl und Liste neuer Organisationen, Venues, Spaces, Events, User, Partneranfragen, Mitgliedschaften und Bilder | Datensatz öffnen, prüfen, als gesichtet markieren |
| Dringend | Fehler an veröffentlichten oder bald stattfindenden Veranstaltungen; fehlende oder widersprüchliche Beziehungen | Betroffenen Datensatz mit genauer Fehlerbeschreibung öffnen |
| Offene Vorgänge | Ausstehende Partneranfragen, alte Teameinladungen, lange nicht aktivierte Accounts | Vorgang prüfen und bestehende Verwaltungsaktion aufrufen |
| Datenqualität | Venues ohne Geoposition, ungültige URLs, unvollständige Inhalte, Bildprobleme | Gefilterte Liste abarbeiten |
| Prüfstatus | Letzter erfolgreicher Prüflauf, fehlgeschlagene Prüfungen, noch ungeprüfte Datensätze | Aktualität und Abdeckung beurteilen |

Zeiträume: heute, letzte 24 Stunden, letzte 7 Tage, frei wählbar. „Heute“ ist ein Kalendertag in der festgelegten Admin-Zeitzone; es ist nicht dasselbe wie „letzte 24 Stunden“.

Filter: Objektart, Organisation, Schweregrad, Regel, Veröffentlichungsstatus, bevorstehender Termin und Bearbeitungsstatus. Listen sollten Suche, stabile Sortierung, Pagination und optional CSV-Export erhalten.

Ein Befund enthält mindestens Objektname, Organisation, Regel, verständlichen Fehlergrund, betroffenes Feld, ersten/letzten Fundzeitpunkt, Priorität und Bearbeitungslink. Beispiel: „Venue Hafenbühne: Geoposition fehlt; betroffen sind drei kommende Termine.“ Das ist ein Anzeigeentwurf, kein Befund aus vorhandenen Daten.

## 2. Neue Datensätze und Aktivitäten

| Objekt | Quelle für „neu“ | Weitere sinnvolle Angaben |
| --- | --- | --- |
| Organisation | `organization.created_at` | Name, Ersteller, Kontakt, Anzahl Venues und Teammitglieder |
| Venue | `venue.created_at` | Organisation, Stadt, Geoposition, Ersteller |
| Space | `space.created_at` | Venue, Organisation, Kapazität |
| Event | `event.created_at` | Organisation, Veröffentlichungsstatus, nächster Termin |
| User | `user.created_at` | Aktivierungsstatus, Teamzuordnungen |
| Partneranfrage | `organization_partner_request.created_at` | Absender, Empfänger, anfragender User, Status, Alter |
| Teammitgliedschaft | `organization_member_link.created_at` | Organisation, User, Einladung, `has_joined`, Rechtezuordnung |
| Bild | `pluto_image.created_at` | Vorschau, Ersteller, Verwendungsorte, Metadaten |

Zusätzlich neue Veranstaltungstermine über `event_date.created_at` anzeigen. Ein altes Event kann neue Termine bekommen, ohne als neu angelegtes Event aufzutauchen.

Die vorhandenen Daten haben Grenzen:

- `pluto_image.created_at` ist nullable. Fehlende Zeitstempel separat ausweisen; deren Reihenfolge lässt sich nicht zuverlässig rekonstruieren.
- `pluto_image_link` hat keinen Zeitstempel. Ein neues Bild und eine neue Verwendung eines vorhandenen Bildes sind unterschiedliche Aktivitäten. Letztere ist historisch derzeit nicht datierbar.
- Mitgliedschafts-`created_at` ist nicht der Annahmezeitpunkt einer Einladung. `invited_at` und `has_joined` existieren, ein eigenes `joined_at` fehlt.
- Partneranfragen haben keinen Änderungs- oder Entscheidungszeitpunkt. Der Ablehnungs-Handler löscht die Anfrage; eine Ablehnungshistorie ist dadurch nicht verfügbar.
- Rechte- und Grant-Tabellen besitzen keine Erstellungs-/Änderungszeitstempel.
- `modified_at` zeigt die letzte Änderung, keine Änderungshistorie. Insbesondere ist `user.modified_at` kein belegter letzter Login. Die bestehende Teamabfrage benennt diesen Wert dennoch als `last_active_at` und Mitgliedschafts-`created_at` als `joined_at`; diese Bezeichnungen sollte das Dashboard nicht übernehmen.

Für „seit meinem letzten Besuch“ einen persönlichen Sichtungsstand speichern; für „bereits geprüft“ eine explizite Sichtungsmarkierung pro Objekt und gegebenenfalls Objektversion. Mittel- bis langfristig ein Aktivitätsprotokoll für Neuanlage, Änderung, Löschung, Einladung, Annahme, Partnerentscheidung und neue Bildzuordnung ergänzen. Historische Vorgänge dürfen daraus nicht rückwirkend erfunden werden.

Quellen: [DDL-Verzeichnis](../ddl/), [Teamabfrage](../sql/admin-get-org-members.sql), [Partneranfragen-Handler](../api/admin_insert_org_partner_request.go).

## 3. Empfohlener Prüfkatalog

„Ungültig“ sollte aus konkreten Regeln entstehen. Drei Stufen sind sinnvoll: Fehler für widersprüchliche oder nicht nutzbare Daten, Warnung für wahrscheinlichen Handlungsbedarf, Hinweis für optionale Ergänzungen. Fehlende Inhalte können bei einem Entwurf erlaubt sein und bei einer bevorstehenden Veröffentlichung dringend werden.

| Objekt | Prüfungen | Einordnung |
| --- | --- | --- |
| Organisation | Name leer oder nur Leerzeichen; ungültiger Weblink; fehlende Kontaktmöglichkeit; kein beigetretenes, aktives Mitglied mit erforderlichen Verwaltungsrechten; Selbstbezug oder Zyklus in der übergeordneten Organisation; ungültige Einträge in `member_of_orgs` | Leerer Name/ungültige Beziehung: Fehler; Kontakt/Betreuung: Warnung. JSON-Struktur zuerst fachlich klären. |
| Venue | Geoposition fehlt/ist leer; Koordinaten außerhalb gültiger Bereiche; unvollständige Adresse; ungültige Web-/Ticketlinks; Schließdatum vor Eröffnung; kommende Veranstaltung nach Schließung | Koordinatenfehler: Fehler; fehlender Punkt: Warnung mit höherer Priorität bei kommenden Terminen. |
| Space | Leerer Name; ungültiger Weblink; negative Fläche/Kapazität; Sitzplätze über Gesamtkapazität; unbekannter Raumtyp | Wertewidersprüche: Fehler; unbekannter Typ gegen vereinbarte Werteliste prüfen. |
| Event | Leerer Titel; keine Termine; ungültige Links; fehlender wirksamer Ort und kein gültiger Online-Link; Raum gehört nicht zum wirksamen Ort; Alter/Preis von–bis widersprüchlich; fehlende Typzuordnung, Beschreibung oder Hauptbild | Ort und Status je Termin auflösen. Fehlender Space allein ist kein Fehler. Inhaltliche Vollständigkeit nach Status priorisieren. |
| Event-Termin | `event_uuid` fehlt; Ende vor Beginn; negative Dauer; widersprüchliche Venue-/Space-Overrides; ungültiger Ticketlink; unplausible Statuskombination | Ganztägige Termine, fehlende Uhrzeiten und Termine über Mitternacht berücksichtigen. Vergangene Termine sind normal. |
| User | Lange nicht aktiviert; E-Mail syntaktisch auffällig; mögliche Dubletten nach vereinbarter Normalisierung; aktiver User ohne erwartete Zuordnung | Keine automatische Fehlerwertung für User ohne Team: reine Besucher-/Favoritenkonten können beabsichtigt sein. Keine Login-Inaktivität aus `modified_at` ableiten. |
| Partneranfrage | Fehlende Organisation/User; Anfrage an sich selbst; unbekannter Status; lange `pending`; akzeptierte Anfrage ohne passende Verbindung | Altersschwelle konfigurierbar. „Accepted ohne Grant“ zunächst als Prüfhinweis, da eine Verbindung später entfernt worden sein könnte. |
| Teammitgliedschaft/Rechte | Lange offene Einladung; beigetretenes Mitglied ohne Rechtezuordnung; Rechtezuordnung ohne passende Mitgliedschaft; doppelte Rechtezeilen; Organisation ohne betreuungsfähiges Team | Nullrechte können beabsichtigt sein. Effektive Rechte und Organisationsbezug berücksichtigen; Einladung und Beitritt getrennt anzeigen. |
| Bild | Bildverknüpfung ohne Bild; unbekannter Kontext/Identifier; Zielobjekt fehlt; Bild ohne Verwendung; fehlende Datei; abgelaufenes Ablaufdatum; ungültige Abmessungen; fehlender Alt-Text oder Herkunfts-/Lizenzangaben | Verwaiste Uploads mit Schonfrist. Dateiabruf separat prüfen. Fehlende Rechteangaben bedeuten Klärungsbedarf, keinen automatisch nachgewiesenen Rechtsverstoß. |

Organisationsübergreifende Venues und Partnerrechte sind vorgesehen. `event.org_uuid != venue.org_uuid` ist daher für sich kein Fehler. Die effektive Berechtigung beziehungsweise freigegebene Nutzung ist entscheidend.

Die bestehende öffentliche Abfrage erbt Venue, Space und Veröffentlichungsstatus aus Event/Termin. Deshalb jeden Termin prüfen: Eine Venue an nur einem Termin darf fehlende Orte an weiteren Terminen nicht verdecken. Wird die Venue überschrieben, muss ein gegebenenfalls geerbter Space noch dazu passen. Ein nicht leerer Online-Link sollte erst nach erfolgreicher URL-Prüfung als gültige Online-Alternative zählen.

Optionale weitere Regeln: mögliche Dubletten bei Organisationen/Venues/Events; ungültige Klassifikationswerte in Arrays und JSON; fehlende oder fachlich abweichende Event-Projektionen. Dubletten anhand mehrerer Merkmale markieren, nicht automatisch zusammenführen. Projektionen sind abgeleitete Daten und zählen nicht als neue fachliche Objekte.

Quellen: [Event](../ddl/event.ddl), [Termine](../ddl/event_date.ddl), [Venue](../ddl/venue.ddl), [öffentliche Event-Abfrage](../sql/get-events-projected.sql), [effektive Organisationsrechte](../sql/admin-get-user-org-permissions.sql).

## 4. Venues ohne Geoposition

Die Position steht in `venue.point` als `geometry(Point,4326)`. Ein einfacher erster Prüfschritt ist:

```sql
SELECT v.uuid, v.name, v.org_uuid, o.name AS organization_name,
       v.street, v.house_number, v.postal_code, v.city, v.country
FROM uranus.venue AS v
JOIN uranus.organization AS o ON o.uuid = v.org_uuid
WHERE v.point IS NULL OR ST_IsEmpty(v.point)
ORDER BY v.city NULLS LAST, v.name, v.uuid;
```

Weitere Regeln: Längengrad außerhalb −180 bis 180, Breitengrad außerhalb −90 bis 90; Punkt unplausibel zur Adresse beziehungsweise zum Land. Der Geometrietyp mit SRID allein garantiert keine plausible Position. `(0, 0)` ist geometrisch gültig und sollte nur als Verdachtsfall gelten. Eine Karte hilft bei falschen Positionen; fehlende Positionen brauchen eine Liste.

Für die Priorisierung die Anzahl kommender betroffener Termine mit deren effektivem Ort ermitteln. Geocoding kann später Korrekturvorschläge liefern; mehrdeutige Adressen sollten vor dem Speichern geprüft werden.

## 5. URLs ohne HTTP oder HTTPS

Zuerst die fachlichen Quellfelder prüfen:

- `organization.web_link`
- `venue.web_link`, `venue.ticket_link`
- `space.web_link`
- `event.source_link`, `online_link`, `ticket_link`, `registration_link`
- `event_date.ticket_link`
- `event_link.url`
- Zusätzlich `license.url` als Prüfung der Stammdaten.

`venue.wikipedia` und `venue.wikidata` erst nach Klärung des Feldformats aufnehmen: Ein Wikidata-Identifier wie `Q123` ist keine fehlerhafte URL, wenn dort ein Identifier erwartet wird. E-Mail- und Telefonfelder sind ebenfalls keine HTTP-URLs. Projektionen nicht als zusätzliche Fehlerquellen zählen; gegebenenfalls gesondert auf Abweichungen prüfen.

Beispiel für alle oben genannten eindeutigen URL-Felder:

```sql
WITH urls AS (
    SELECT 'organization'::text AS entity_type, uuid::text AS entity_key,
           'web_link'::text AS field, web_link AS value
    FROM uranus.organization
    UNION ALL
    SELECT 'venue', v.uuid::text, x.field, x.value
    FROM uranus.venue v
    CROSS JOIN LATERAL (VALUES
        ('web_link', v.web_link), ('ticket_link', v.ticket_link)
    ) AS x(field, value)
    UNION ALL
    SELECT 'space', uuid::text, 'web_link', web_link FROM uranus.space
    UNION ALL
    SELECT 'event', e.uuid::text, x.field, x.value
    FROM uranus.event e
    CROSS JOIN LATERAL (VALUES
        ('source_link', e.source_link), ('online_link', e.online_link),
        ('ticket_link', e.ticket_link), ('registration_link', e.registration_link)
    ) AS x(field, value)
    UNION ALL
    SELECT 'event_date', uuid::text, 'ticket_link', ticket_link FROM uranus.event_date
    UNION ALL
    SELECT 'event_link', id::text, 'url', url FROM uranus.event_link
    UNION ALL
    SELECT 'license', key::text, 'url', url FROM uranus.license
)
SELECT entity_type, entity_key, field, value
FROM urls
WHERE value IS NOT NULL
  AND value !~ '^[[:space:]]*$'
  AND value !~* '^[[:space:]]*https?://'
ORDER BY entity_type, entity_key, field;
```

Das ist nur eine Suche nach fehlendem HTTP(S)-Präfix, keine vollständige URL-Validierung. `https://` ohne Host, eingebettete Leerzeichen und weitere Syntaxfehler brauchen zusätzlich einen URL-Parser. Leere Pflichtlinks separat prüfen; optionale fehlende Links sind nicht automatisch Fehler. Groß-/Kleinschreibung des Schemas berücksichtigen und umgebende Leerzeichen als Normalisierungsbedarf behandeln.

Für die Anwendung gibt es bereits `ValidateOptionalUrl` in [app/validate.go](../app/validate.go). Die Repo-Suche zeigt Aufrufe für Source-/Online-Link beim Erstellen eines Events, aber keine flächendeckende Verwendung. Beispielsweise schreiben die untersuchten Feld-Updates von Organisation/Venue und das Event-Link-Update ohne diesen Validator. Eine gemeinsame, konsistent verwendete Validierung sollte neue Probleme vermeiden. Der aktuelle Validator prüft das Präfix case-sensitiv; Normalisierung und akzeptierte Schreibweisen deshalb einheitlich festlegen.

Erreichbarkeit ist eine eigene Prüfung: HTTP-Status, Redirect-Ziel, letzter Versuch und letzter Erfolg speichern; Timeouts und vorübergehende Fehler wiederholen. Ein 403 oder 429 beweist keinen kaputten Link. Solche Prüfungen zeitgesteuert im Hintergrund mit Limits ausführen. Da gespeicherte URLs fremde Eingaben sind, nur öffentliche HTTP(S)-Ziele abrufen und interne/private Ziele auch nach DNS-Auflösung und Redirects ausschließen. Niemals automatisch pauschal `https://` vor beliebige Werte setzen.

Die SQL-Beispiele sind aus dem DDL abgeleitet und wurden nicht gegen eine Datenbank ausgeführt.

## 6. Konkrete Auffälligkeiten im Schema und vorhandene Ansätze

1. **Partneranfragen:** `from_org_uuid`, `to_org_uuid`, `from_user_uuid` besitzen im DDL keine Foreign Keys. `status` ist unbeschränkter Text. Fehlende Ziele und unbekannte Statuswerte sind deshalb technisch möglich. Ein stabiler technischer Primärschlüssel und Entscheidungszeitpunkte wären für ein Vorgangsprotokoll hilfreich. Aktuell identifiziert das eindeutige Organisationspaar die Anfrage.
2. **Richtung der Partnerschaft:** Die Annahme von A → B legt im Handler den Grant mit `src_org_uuid = B`, `dst_org_uuid = A` an. Eine Konsistenzprüfung muss genau diese Richtung berücksichtigen. Der neue Grant erhält bewusst `permissions = 0`; das ist allein kein Fehler.
3. **Bilder:** `pluto_image_link.context_uuid` ist eine polymorphe Verknüpfung ohne Foreign Key zum fachlichen Ziel. `pluto_image_uuid` ist nullable. Deshalb Kontext, Identifier, Ziel und Bild separat prüfen. Der generische Upload-Handler unterstützt Organisation, Venue, Event und Portal; Space ist dort auskommentiert. Keine pauschale Pflichtbildregel für Spaces einführen.
4. **Teamrechte:** Mitgliedschaft und Berechtigung liegen getrennt in `organization_member_link` und `user_organization_link`. Die verschiedenen `user_*_link`-Tabellen haben im DDL keine eindeutige Paar-Constraint. Doppelte Zuordnungen sind möglich und können Auswertungen oder die Rechteauflösung beeinflussen.
5. **Termine:** `event_date.event_uuid` hat einen Foreign Key, erlaubt aber NULL. Ein Termin ohne Event ist dadurch technisch möglich.
6. **Venue-Scope:** `venue.scope` hat den Default `'standard'`, die CHECK-Constraint erlaubt aber nur `'organization'` und `'shared'`. Wird der Default verwendet, widerspricht er der Constraint. Das ist ein konkreter Widerspruch im DDL, der gegen das laufende Schema zu prüfen ist.
7. **Space-Features:** `space_feature_link.space_id` ist Integer ohne FK zum Space, während `space.uuid` UUID verwendet. Klären, ob dies ein Altbestand ist oder ein Zuordnungsmodell fehlt.
8. **Zeitangaben:** Die relevanten Erstellungsdaten verwenden überwiegend `timestamp without time zone`. Vor Zeitraumfiltern die Speicherzeitzone klären. Neue Auditdaten vorzugsweise mit eindeutigem Zeitpunkt speichern; bestehende Werte nicht ohne geklärte Semantik umdeuten.
9. **Bestehende Event-Prüfung:** [admin-get-user-event-notifications.sql](../sql/admin-get-user-event-notifications.sql) prüft unter anderem Bild, Termine, Ort/Online-Link, Typ und Titel. Sie filtert jedoch auf `draft`/`review` und eine Terminvorschau beziehungsweise terminlose Events. Veröffentlichte problematische Events werden so nicht vollständig erfasst. Außerdem genügt dort bereits irgendein Termin mit Venue, und für ein Bild wird nur die Linkzeile geprüft. Das ist ein brauchbarer Ausgangspunkt, aber noch kein vollständiger Qualitätscheck.
10. **Projektionen:** `event_projection` und `event_date_projection` werden durch vorhandene Refresh-Funktionen aktualisiert. Qualitätsregeln auf den Quelltabellen berechnen; fehlende oder abweichende Projektionen gesondert prüfen. Zeitstempel allein beweisen wegen der Abhängigkeiten zu Venue, Space, Organisation und Bildern keine vollständige Aktualität.

Die vorhandenen Foreign Keys verhindern bei aktiv durchgesetztem Schema bereits viele echte verwaiste Beziehungen, etwa Space → Venue. Dennoch können NULL-Werte nach `ON DELETE SET NULL` fachliche Lücken erzeugen. Der Schwerpunkt sollte daher auf nicht abgesicherten Verknüpfungen, Nullbeziehungen und widersprüchlichen Kombinationen liegen.

## 7. Technische Umsetzung

Die README beschreibt ein Go-Backend, PostgreSQL/PostGIS, Pluto und ein separates Vue3-Frontend `uranus-dashboard`. Empfehlung: Qualitätsregeln und APIs hier im Backend implementieren und die Verwaltungsoberfläche im bestehenden Frontend ergänzen. Das Frontend-Repository wurde in dieser Untersuchung nicht geprüft.

Für den Einstieg sind SQL-Abfragen beziehungsweise Views für Aktivitäten und deterministische Datenregeln ausreichend. Über Go-Endpunkte werden gefilterte Summen und paginierte Befundlisten bereitgestellt. Keine Browser-Direktverbindung zur Datenbank. Einen zusätzlichen Analyse-Dienst braucht dieser erste Umfang nicht.

Alle Regeln liefern ein gemeinsames Befundformat, beispielsweise:

```text
rule_code, rule_version, entity_type, entity_key, org_uuid,
field, severity, message, first_seen_at, last_seen_at,
last_checked_at, resolved_at, workflow_status, assigned_to, snoozed_until
```

`entity_key` muss auch zusammengesetzte Schlüssel unterstützen: Partneranfrage = Organisationspaar, Mitgliedschaft = Organisation + User. Ein Befund wird über Regel, Objekt und gegebenenfalls Feld/Unterobjekt stabil identifiziert, damit wiederholte Scans keine Duplikate erzeugen.

Für Abarbeitung eine neue Befund-/Review-Ablage vorsehen. „Offen“, „in Bearbeitung“, „zurückgestellt“ und „begründete Ausnahme“ sind menschliche Entscheidungen. „Behoben“ bestätigt eine erneute erfolgreiche Prüfung. Befunde nur dann automatisch schließen, wenn die betreffende Regel vollständig und erfolgreich geprüft wurde; ein ausgefallener Prüflauf bedeutet nicht Fehlerfreiheit. Ausnahmen bei relevanten Datenänderungen erneut bewerten.

Die vorhandene `todo`-Tabelle kann persönliche Aufgaben ergänzen, hat aber keine Regelkennung oder Objektverknüpfung und ersetzt daher diese Befundablage nicht.

Zu Beginn regelmäßige Datenbankprüfungen, beispielsweise alle 15–60 Minuten, und manuelles Aktualisieren anbieten. Nach Änderungen betroffene Objekte und deren abhängige Events erneut prüfen; periodische Gesamtläufe fangen ausgelassene Änderungen auf. HTTP-/Dateiprüfungen separat mit geringerer Frequenz und eigenem Prüfstatus betreiben. Die Intervalle sind Startvorschläge und anhand Datenmenge und Laufzeit anzupassen.

Indizes anhand tatsächlicher Abfragepläne und Datenvolumen ergänzen, insbesondere für Zeitfilter, offene Vorgänge und Joins. Erst bei Bedarf Ergebnisse voraggregieren. Für Aktivitäten einen stabilen Cursor aus Zeitpunkt und Objektschlüssel verwenden.

Ein organisationsübergreifendes Dashboard braucht eine explizite, serverseitig geprüfte Systemadministrator-Berechtigung. Ein gültiges Login ist keine globale Administrationsberechtigung. Vorhandene Organisationsrechte weiter für entsprechend begrenzte Sichten nutzen. Passwort-Hashes und Aktivierungs-, Einladungs- oder Importtokens gehören nicht in die Dashboard-Antworten. Korrekturen durch vorhandene autorisierte Schreibpfade mit Validierung, Projektionserneuerung und Änderungsprotokoll ausführen.

Quellen: [README](../README.md), [JWT-Middleware](../app/middleware.go), [Todo-Schema](../ddl/todo.ddl), [Projektionserneuerung](../api/refresh_event_projections.go).

## 8. Empfohlene Reihenfolge

1. **Grundlagen abgleichen:** Live-Schema und Zeitzone, erlaubte Status-/Kontextwerte, globale Adminrechte sowie Fehler-/Warnregeln festlegen. Insbesondere DDL-Widersprüche verifizieren.
2. **Erste nutzbare Version:** Neue Einträge für alle acht Bereiche plus Termine; Venues ohne Geoposition; URL-Syntax; Events ohne Termine oder wirksamen Ort; Raum-/Venue-Widersprüche; offene Partneranfragen und Einladungen; fehlende Bildziele. Filter und direkte Bearbeitungslinks gehören dazu.
3. **Tägliche Abarbeitung:** Sichtungsstand, Befundstatus, Zuweisung, begründete Ausnahmen, Priorisierung nach Veröffentlichung und bevorstehenden Terminen, Anzeige letzter erfolgreicher Prüfungen.
4. **Vertiefung:** Bilddateien und URLs auf Erreichbarkeit prüfen, Dublettenvorschläge, Geocoding-Vorschläge, Projektionen vergleichen, Aktivitätsprotokoll und passende Datenbank-Constraints ergänzen.

Die erste Version ist gut abgrenzbar und erfordert keine grundlegende Neugestaltung des Datenmodells. Der größere fachliche Aufwand liegt in verlässlichen Regeln mit wenigen Fehlalarmen, korrekten Berechtigungen und der Integration in die Bearbeitungsoberfläche. Eine belastbare Aufwandsschätzung setzt zusätzlich die Prüfung des Frontends und der tatsächlichen Datenbank voraus.
