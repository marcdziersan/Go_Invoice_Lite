# GoInvoice Lite

GoInvoice Lite ist eine kleine Business-Webanwendung in Go zur Verwaltung von Kunden, Angeboten und Rechnungen. Die Anwendung setzt bewusst auf serverseitiges Rendering, einfache HTTP-Routen, Sessions, JSON-Persistenz und eine klare Projektstruktur ohne zusätzliches Framework.

## Kurzbeschreibung

Kleines Business-MVP in Go für Kundenverwaltung, Angebote, Rechnungen, Dashboard-Kennzahlen und CSV-Export.

## Funktionsumfang

- Login mit Session-Cookie
- Kunden anlegen, bearbeiten, anzeigen und löschen
- Angebote anlegen, bearbeiten, anzeigen und löschen
- Angebote in Rechnungen umwandeln
- Rechnungen anlegen, bearbeiten, anzeigen und löschen
- Statusverwaltung für Angebote und Rechnungen
- Dashboard mit einfachen Kennzahlen
- CSV-Export für Rechnungen
- JSON-basierte Datenspeicherung
- Admin-Benutzer beim ersten Start

## Technischer Ansatz

Das Projekt verwendet die Go Standard Library und verzichtet bewusst auf große externe Abhängigkeiten.

Verwendete Bausteine:

- `net/http` für Routing und HTTP-Handling
- `html/template` für serverseitige Templates
- `encoding/json` für Persistenz
- `encoding/csv` für CSV-Export
- `crypto/hmac`, `crypto/sha256` und PBKDF2-ähnliche Ableitung für Passwort-Hashing
- einfache Session-Verwaltung über HTTP-Cookies

## Projektstruktur

```txt
Go_Invoice_Lite/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   ├── auth/
│   ├── customer/
│   ├── dashboard/
│   ├── invoice/
│   ├── quote/
│   └── storage/
├── web/
│   ├── static/
│   │   └── style.css
│   └── templates/
├── data/
│   └── app.json
├── go.mod
└── README.md
```

## Installation

Voraussetzung ist eine installierte Go-Version.

```bash
git clone https://github.com/marcdziersan/Go_Invoice_Lite.git
cd Go_Invoice_Lite
go run ./cmd/server
```

Danach ist die Anwendung unter folgender Adresse erreichbar:

```txt
http://localhost:8080
```

## Standard-Login

Beim ersten Start wird ein Admin-Benutzer angelegt.

```txt
Benutzername: admin
Passwort: admin123
```

Der Standard-Zugang ist nur für die lokale Entwicklung vorgesehen und sollte bei einer Weiterentwicklung ersetzt oder mindestens geändert werden.

## Datenhaltung

Die Anwendung speichert ihre Daten lokal in einer JSON-Datei.

```txt
data/app.json
```

Die Persistenz ist bewusst einfach gehalten. Für eine spätere Erweiterung kann die Speicherlogik durch SQLite, PostgreSQL oder MySQL ersetzt werden.

## Zentrale Bereiche

### Kunden

Kunden bilden die Grundlage für Angebote und Rechnungen. Erfasst werden Stammdaten wie Firma, Ansprechpartner, E-Mail, Telefon, Adresse und Notizen.

### Angebote

Angebote besitzen eine Angebotsnummer, einen Kundenbezug, Beträge, Steuerinformationen und einen Status. Ein angenommenes Angebot kann in eine Rechnung umgewandelt werden.

### Rechnungen

Rechnungen besitzen eine Rechnungsnummer, einen Kundenbezug, Beträge, Steuerinformationen, Rechnungsdatum, Fälligkeitsdatum und Status.

### Dashboard

Das Dashboard zeigt einfache Kennzahlen wie Anzahl der Kunden, Angebote, Rechnungen, offene Beträge und bezahlte Umsätze.

### CSV-Export

Rechnungsdaten können als CSV-Datei exportiert werden. Dadurch lassen sich Daten einfach weiterverarbeiten oder kontrollieren.

## Beispielhafte Routen

```txt
GET  /login
POST /login
POST /logout
GET  /dashboard
GET  /customers
GET  /customers/new
POST /customers/new
GET  /customers/edit
POST /customers/edit
POST /customers/delete
GET  /quotes
GET  /quotes/new
POST /quotes/new
GET  /quotes/edit
POST /quotes/edit
POST /quotes/delete
POST /quotes/convert
GET  /invoices
GET  /invoices/new
POST /invoices/new
GET  /invoices/edit
POST /invoices/edit
POST /invoices/delete
GET  /invoices/export.csv
```

## Sicherheitshinweise

Die Anwendung enthält grundlegende Sicherheitsmaßnahmen für ein lokales MVP:

- Passwörter werden nicht im Klartext gespeichert
- Session-Cookies werden serverseitig validiert
- geschützte Seiten sind nur nach Login erreichbar
- Formulare arbeiten mit serverseitiger Verarbeitung
- Beträge werden intern als Ganzzahlen verarbeitet

Für einen produktiven Einsatz wären weitere Maßnahmen notwendig, zum Beispiel:

- CSRF-Schutz
- HTTPS-Konfiguration
- sichere Session-Persistenz
- Passwortwechsel und Benutzerverwaltung
- Datenbank mit Transaktionen
- serverseitige Eingabevalidierung mit detaillierten Fehlermeldungen
- rollenbasierte Rechteprüfung
- Audit-Log
- Backup-Strategie

## Mögliche Weiterentwicklung

- SQLite- oder PostgreSQL-Persistenz
- Benutzerverwaltung
- Passwort ändern/zurücksetzen
- PDF-Export für Rechnungen
- Positionslisten für Angebote und Rechnungen
- Steuersätze pro Position
- Zahlungsdatum und Mahnstufen
- Such- und Filterfunktionen
- API-Endpunkte
- automatisierte Tests
- Migrationssystem
- Audit-Log

## Entwicklungsbefehle

Projekt starten:

```bash
go run ./cmd/server
```

Tests ausführen:

```bash
go test ./...
```

Go-Code formatieren:

```bash
gofmt -w .
```

## Lizenz

Dieses Projekt kann unter der MIT-Lizenz veröffentlicht werden. Eine passende `LICENSE`-Datei sollte bei Bedarf ergänzt werden.
