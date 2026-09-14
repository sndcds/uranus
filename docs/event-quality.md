# Qualitätsprüfung von Events und Terminen

`GET /api/admin/event/:eventUuid/quality`

Der Endpunkt bewertet die gespeicherten Eventdaten einschließlich aller Termine und verknüpften Bilder. Er liefert redaktionelle Hinweise auf Deutsch und verändert keine Daten oder Freigaben. Ein Bearer-Token oder das bestehende Access-Token-Cookie ist erforderlich. Der Benutzer benötigt beim Veranstalter mindestens eines der Rechte `UserPermEditEvent` oder `UserPermViewEventInsights`, wie beim Abruf der Eventdetails.

```sh
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$API_BASE/api/admin/event/$EVENT_UUID/quality"
```

Die Antwort verwendet die bestehende API-Hülle. Unter `data` stehen:

| Feld | Bedeutung |
| --- | --- |
| `event_uuid` | Geprüftes Event |
| `rules_version` | Version der Regeln, zunächst `1` |
| `status` | `ok`, `recommendations` (Empfehlungen oder Tipps), `warnings` (mindestens eine Warnung) |
| `completeness` | `score` von 0–100, Anzahl vorhandener (`present`) und geprüfter (`total`) Angaben |
| `metrics` | Titelzeichen, Textstatistiken für Beschreibung/Zusammenfassung, Bild- und Terminanzahl |
| `issues` | Hinweise als Array; bei unauffälligen Daten `[]` |

Jeder Hinweis enthält einen stabilen `code`, `severity` (`warning`, `recommendation`, `info`), `field`, `message` und eine konkrete `recommendation`. Termin- und Bildhinweise enthalten zusätzlich `event_date_uuid` bzw. `image_uuid`. Manche Regeln liefern in `details` Messwerte und Grenzwerte, Bildhinweise auch den Bild-Identifier. `field` bezeichnet das Feld im Admin-Event; die UUID identifiziert das Element einer Liste. Beim fehlenden Terminort bezeichnet `dates.venue_uuid` den Ort einschließlich der möglichen Alternativen am Event.

Beispiel eines Eintrags in `data.issues`:

```json
{
  "code": "image.low_resolution",
  "severity": "warning",
  "field": "images",
  "image_uuid": "0198b7c0-0000-7000-8000-000000000001",
  "message": "Das Bild hat eine geringe Auflösung.",
  "recommendation": "Ein Bild mit mindestens 1200 Pixeln an der langen und 630 Pixeln an der kurzen Seite verwenden.",
  "details": {
    "identifier": "main",
    "width": 640,
    "height": 480,
    "min_long_edge": 1200,
    "min_short_edge": 630
  }
}
```

## Vollständigkeit

Sechs Angaben am Event werden gleich gewichtet: Titel, Beschreibung, Zusammenfassung, mindestens ein Bild, mindestens eine Veranstaltungsart und mindestens ein Termin. Pro Termin kommen Startdatum und Ort/Online-Link hinzu; bei nicht ganztägigen Terminen zusätzlich die Startzeit. Der Ort darf vom Event geerbt werden; ein Online-Link am Event genügt ebenfalls.

`score = round(100 × present / total)`. Ein Event ohne Angaben hat 0 Prozent. Ein vollständig ausgefülltes Event mit einem regulären Termin hat 9 von 9 Angaben, mit einem ganztägigen Termin 8 von 8. Leere bzw. reine Whitespace-Texte und leeres HTML zählen als fehlend. Bildmaße zählen nicht zur Anwesenheit eines Bildes.

Der Wert beschreibt ausschließlich vorhandene Angaben. Auch bei 100 Prozent können Qualitätswarnungen vorliegen, beispielsweise wegen zu kleiner Bilder oder eines Enddatums vor dem Startdatum. Termine werden einzeln gewichtet; der Wert ist deshalb kein gewichteter redaktioneller Qualitätsscore.

## Regeln in Version 1

Die Grenzwerte sind erste redaktionelle Heuristiken und zentral in `service/event_quality.go` definiert. Grenzwerte selbst sind zulässig.

| Bereich | Prüfung | Schweregrad |
| --- | --- | --- |
| Titel | Fehlt | Warnung |
| Titel | Weniger als 10 oder mehr als 120 Zeichen | Empfehlung |
| Beschreibung | Fehlt | Warnung |
| Beschreibung | Weniger als 200 Zeichen oder 40 Wörter | Empfehlung |
| Beschreibung | Höchstens ein erkennbarer Satz | Empfehlung |
| Beschreibung | Mehr als 10.000 Zeichen, 20 Textblöcke oder 8 Überschriften | Je eine Empfehlung |
| Zusammenfassung | Fehlt | Warnung |
| Zusammenfassung | Weniger als 80 oder mehr als 500 Zeichen | Empfehlung |
| Zusammenfassung | Mehr als ein Textblock | Empfehlung: keine Absatzumbrüche |
| Zusammenfassung | Mehr als 3 Fett-/Kursiv-Auszeichnungen oder mehr als 25 % hervorgehobener Text | Empfehlung |
| Zusammenfassung | Weitere Formatierungen, z. B. Überschriften, Listen, Links, Code, Durchstreichungen, Bilder, eigene CSS-Stile | Empfehlung |
| Bilder | Kein Bild verknüpft | Warnung |
| Jedes Bild | Lange Seite kleiner als 1200 oder kurze Seite kleiner als 630 Pixel | Warnung, unabhängig von Hoch-/Querformat |
| Jedes Bild | Breite/Höhe fehlt oder ist nicht positiv | Tipp: Auflösung unbekannt |
| Veranstaltungsart | Fehlt | Warnung |
| Termine | Kein Termin vorhanden | Warnung |
| Jeder Termin | Startdatum fehlt; Startzeit fehlt bei nicht ganztägigen Terminen | Je eine Warnung |
| Jeder Termin | Weder eigener/geerbter Veranstaltungsort noch Online-Link am Event | Warnung |
| Jeder Termin | Ungültiges angegebenes Datum oder Uhrzeit | Warnung |
| Jeder Termin | Enddatum vor Startdatum; bei gleicher Tageszuordnung Endzeit nicht nach Startzeit (außer ganztägig) | Warnung |
| Jeder Termin | Angegebene Dauer ist null oder negativ | Warnung |
| Jeder nicht ganztägige Termin | Weder Endzeit noch Dauer angegeben | Tipp |

Die Texte werden als Markdown (entsprechend dem Dashboard, einschließlich GFM), HTML oder einfacher Text ausgewertet. Gezählt werden Unicode-Codepoints des sichtbaren Textes nach Zusammenfassen von Whitespace; Markdown-Marker, HTML-Tags und Linkziele zählen nicht zur Textlänge. HTML-Entities werden dekodiert. Script-, Style- und Template-Inhalte werden ignoriert. Ein einzelner Absatz-Wrapper ist unauffällig. Textblöcke werden durch Absatz-/Blockelemente, explizite HTML- oder Markdown-Zeilenumbrüche oder Leerzeilen getrennt; einfache Text-Zeilenumbrüche gelten als Zeilenumbruch innerhalb desselben Absatzes. Überschriften zählen ebenfalls als Textblock. Der Anteil hervorgehobener Zeichen wird ohne Whitespace ermittelt, verschachtelte Hervorhebungen zählen beim Anteil nur einmal.

Die Satzzählung ist eine Näherung anhand von `.`, `!`, `?` und entsprechenden CJK-Satzzeichen; Abkürzungen können zusätzliche Sätze ergeben. Eine semantische Prüfung von Textinhalt, Bildschärfe oder Bildmotiven erfolgt nicht. Die Auflösung wird anhand gespeicherter Originalmaße geprüft, ohne Bilder herunterzuladen. Fehlende Auflösungsdaten werden ausdrücklich als unbekannt ausgewiesen.

Alle gespeicherten Termine werden geprüft, auch vergangene. Es gibt zunächst keine Warnung allein aufgrund vergangener Termine und keine Pflicht für Tickets, Preise, Barrierefreiheitsangaben oder optionale Endzeiten. Ein Termin über Mitternacht benötigt ein entsprechendes Enddatum, damit eine frühere Endzeit korrekt zugeordnet werden kann.

## Fehler und Betrieb

- `200`: Prüfung erfolgreich, auch bei Warnungen.
- `400`: Fehlende oder ungültige Event-UUID.
- `401`: Keine gültige Anmeldung.
- `404`: Event existiert nicht oder der Benutzer hat keines der benötigten Rechte; entsprechend dem Eventdetails-Endpunkt.
- `500`: Datenbankfehler. Fehler beim Laden einzelner Beziehungen brechen die Prüfung ab und erzeugen keinen scheinbar vollständigen Bericht.

Keine Datenbankmigration erforderlich. Die bestehende Bildabfrage lädt nun zusätzlich `width` und `height`; dadurch sind diese Werte auch im Admin-Eventdetails-Endpunkt verfügbar. Regeln und Datenmodell liegen getrennt von HTTP und Datenbankzugriff. Einbindung in das Dashboard ist nicht Teil dieses Endpunkts.

Tests: `go test ./...`. Es gibt Tests für die Bewertungsregeln und Grenzwerte, Markdown/HTML, Vererbung, Ganztagstermine, mehrere Termine/Bilder, Termine über Mitternacht, JSON-Ausgabe, ungültige UUIDs und fehlende Anmeldung. Ein Integrationstest gegen eine laufende PostgreSQL-Datenbank ist nicht enthalten.
