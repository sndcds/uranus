# Event-Endpunkte und Lookups: Überblick für Entwickler

Stand: 20.09.2026. Grundlage: [oas3-combined.yaml](../openapi/oas3-combined.yaml). Ergänzende Codebefunde sind ausdrücklich gekennzeichnet; die Endpunkte wurden nicht live getestet.

`GET /api/events` ist der zentrale Endpunkt für die Eventliste. Er liefert Veranstaltungsdaten inklusive Terminen, Veranstalter, Veranstaltungsort und Preisen. Für lesbare, übersetzte Bezeichnungen müssen einige IDs und Codes über zusätzliche Lookups aufgelöst werden. Die dokumentierte Produktionsbasis ist `https://api.kulturbytes.de`.

## Lookups für die Eventdaten

Alle folgenden Endpunkte verwenden `GET`. Die Zugriffspfade beziehen sich auf die jeweilige Lookup-Antwort.

| Event-Feld | Endpunkt | Auflösung |
| --- | --- | --- |
| `event_types[].type_id` | `/api/event/type-genre-lookup` | Typbezeichnung: `data[lang].types[type_id].name`. |
| `event_types[].genre_id` | Derselbe Type-/Genre-Lookup | Genrebezeichnung: `data[lang].types[type_id].genres[genre_id]`. Genres sind ihrem Typ untergeordnet. |
| `categories[]` | `/api/event/category-lookup?lang=de` | Kategoriebezeichnung: `data.de[category_id].name`. |
| `release_status` | `/api/event/release-status-i18n?lang=de` | Statusbezeichnung: `data.i18n.de[release_status]`, etwa `cancelled` → „Abgesagt“. `data.order` enthält die Sortierreihenfolge. |
| `languages[]` | `/api/choosable-languages?lang=de` | Sprachcode mit `data[].id` abgleichen und `name` anzeigen. |
| `currency` | `/api/choosable-currencies?lang=de` | Währungscode mit `data[].id` abgleichen und `name` anzeigen. |
| `venue_country` | `/api/choosable-countries?lang=de` | Ländercode mit `data[].country_code` abgleichen; Bezeichnung in `country_name`. Der Lookup verwendet dreistellige Codes wie `DEU`. |
| `venue_state` | `/api/choosable-states?country-code=DEU` | Falls als Code geliefert: mit `data[].state_code` abgleichen und `state_name` anzeigen. Das Event-Schema legt das Format nicht genauer fest. |

Der Type-/Genre-Lookup liefert mehrere Sprachen in einer Antwort; ein `lang`-Parameter ist dafür nicht dokumentiert. Die gewünschte Sprache wird clientseitig ausgewählt. Für `{ "type_id": 1, "genre_id": 1002 }` ergeben die dokumentierten Beispieldaten auf Deutsch „Konzert“ und „Rock“. Numerische IDs erscheinen in Lookup-Objekten als String-Schlüssel.

Der Kategorie-Lookup akzeptiert mehrere Sprachen als kommaseparierte Liste; ohne Sprachangabe liefert er alle verfügbaren Sprachen. Der Status-Lookup akzeptiert ebenfalls mehrere Sprachen, verwendet ohne Angabe aber Englisch. Die oben genannten sprachabhängigen `choosable-*`-Endpunkte verwenden standardmäßig Englisch. Der States-Endpunkt wird über das Land ausgewählt und hat keinen dokumentierten Sprachparameter.

## Bereits enthaltene Werte und zusätzliche Filterauswahl

`org_name`, `venue_name` und `space_name` stehen direkt im Event, ebenso Adressdaten, Koordinaten, Titel, Texte, Termine, Altersgrenzen, Preisbeträge, Ticketlink und Tags. Für ihre normale Anzeige sind keine weiteren Namens-Lookups erforderlich. Optionale Felder können fehlen oder `null` sein.

Für Auswahlfelder und Filter sind zusätzlich nützlich:

- `GET /api/choosable-orgs`: Veranstalter suchen und deren UUIDs für `org_uuids` verwenden.
- `GET /api/choosable-venues`: Veranstaltungsorte suchen und deren UUIDs für `venue_uuids` verwenden.
- `GET /api/choosable-space-types?lang=de`: Raumtypen für den Filter `space_types` laden; die Einträge enthalten `key`, `name` und `description`.

Die Organisations- und Venue-Auswahl unterstützt Namenssuche sowie geografische Filter. Namen und UUIDs sind für die normale Eventanzeige bereits im Event enthalten.

Die ebenfalls dokumentierten Lookups für Eventanlässe, Venue-Typen, Rechtsformen, Lizenzen und Linktypen haben im beschriebenen `EventResponse` kein entsprechendes Referenzfeld.

## Lücken und ergänzende Codebefunde

Mit der kombinierten YAML allein lassen sich nicht alle Event-Felder vollständig interpretieren:

| Feld | Befund |
| --- | --- |
| `space_accessibility_flags` | Lookup und Flag-Kodierung fehlen in der YAML. Im Code ist `GET /api/accessibility/flags?lang=de` registriert. Der Handler liefert ein direktes Array mit `id`, `topic_id` und `name`, ohne `data`-Envelope. Die Dekodierung des Event-Flag-Feldes muss zusätzlich geklärt werden. |
| `visitor_info_flags` | In der YAML fehlen sowohl ein Lookup als auch die Beschreibung der Flag-Kodierung. |
| `price_type` | Kein aktiver Preisarten-Lookup: Die Route `/api/choosable-price-types` ist auskommentiert. Im Modell stehen `not_specified`, `regular_price`, `free`, `donation` und `tiered_prices`. Für die Anzeige ist eine eigene Übersetzung erforderlich. |
| `image_ai_label` | Die YAML beschreibt das Feld lediglich als String; ein Lookup oder eine Werteliste fehlt. |

Quellen der Codebefunde: [Routenregistrierung](../uranus-api.go), [Accessibility-Handler](../api/get_accessibility_flags.go), [Event-Modell](../model/event.go).

Weitere Unstimmigkeiten in der Spezifikation:

- Der Kategorie-Lookup zeigt in seinen Beispielen andere Zuordnungen für die IDs `2` und `3` als die Beschreibung des Filters `categories`. Für die Anzeige die tatsächlich gelieferten Lookup-Werte verwenden und die Beispieldaten angleichen.
- Der Länder-Lookup definiert dreistellige ISO-Codes, während Organisations- und Venue-Beispiele teilweise zweistellige Codes zeigen. Das Event-Feld `venue_country` ist nur als String beschrieben. Ein direktes Mapping setzt übereinstimmende Codeformate voraus.

## Weitere Event-Endpunkte

| Endpunkt | Zweck |
| --- | --- |
| `GET /api/events` | Gefilterte Eventliste mit Pagination. |
| `POST /api/events/filter` | Eventabfrage mit JSON-Filtern; Listen werden als Arrays übergeben. |
| `GET /api/events/week?week_start=YYYY-MM-DD` | Wochenansicht, nach Tagen gruppiert. `week_start` ist erforderlich. |
| `GET /api/events/type-summary` | Trefferzahlen je Typ und Genre. Bezeichnungen über den Type-/Genre-Lookup ergänzen. `limit` und `offset` werden ignoriert. |
| `GET /api/events/venue-summary` | Anzahl der Veranstaltungstermine je Venue, einschließlich Venue-Name. `limit` und `offset` werden ignoriert. |
| `GET /api/events/geojson` | Veranstaltungsorte mit passenden Events als GeoJSON für die Kartenansicht. Koordinatenreihenfolge: `[longitude, latitude]`. Ohne passende Venues: HTTP 204. |

## Hinweise für die Integration

1. Die benötigten Lookups parallel zur Eventliste laden und pro Sprache zwischenspeichern. Es ist kein Lookup-Aufruf pro Event nötig.
2. Type und Genre gemeinsam über das jeweilige Paar `type_id`/`genre_id` auflösen. Fehlende Übersetzungen und unbekannte IDs im Frontend abfangen.
3. Bei `GET /api/events` Listenfilter kommasepariert übergeben. `event_types` und `genres` dürfen nicht gleichzeitig als Filter verwendet werden. Unbekannte Query-Parameter führen zu HTTP 400.
4. Für weitere Ergebnisse die beiden Cursorwerte `last_event_start_at` und `last_event_date_uuid` gemeinsam übergeben.
5. Ohne `start` beginnt die Eventabfrage laut Spezifikation am aktuellen Datum; für andere Zeiträume `start` und gegebenenfalls `end` explizit setzen.

## Offene Prüfpunkte

- Zuordnungen der Event-Felder zu den Lookups und die Antwortstrukturen prüfen.
- Fehlende Dokumentation für Accessibility, Besucherhinweise, Preisarten und KI-Bildkennzeichnung bewerten.
- Kategoriebeispiele und Länder-Codeformate vereinheitlichen.
- Prüfen, ob die dokumentierten Event-Antworten ohne Envelope und die Lookup-Antworten mit `data`-Envelope dem tatsächlichen API-Verhalten entsprechen.
