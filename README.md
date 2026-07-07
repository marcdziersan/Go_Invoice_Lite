# GoInvoice Lite

GoInvoice Lite ist eine kleine Business-Webanwendung in Go zur Verwaltung von Kunden, Angeboten, Rechnungen, Benutzern und Audit-Ereignissen. Die Anwendung setzt auf serverseitiges Rendering, einfache HTTP-Routen, Sessions, JSON-Persistenz, ein Migrationssystem und eine schlanke API ohne großes Framework.

## Kurzbeschreibung

Kleines Business-MVP in Go für Kundenverwaltung, Angebote, Rechnungen, Positionslisten, Benutzerverwaltung, Dashboard-Kennzahlen, PDF-/CSV-Export, API-Endpunkte und Audit-Log.

## Funktionsumfang

- Login mit Session-Cookie
- Benutzerverwaltung für Administratoren
- Passwort ändern für angemeldete Benutzer
- Passwort zurücksetzen über die Benutzerverwaltung
- Kunden anlegen, bearbeiten, anzeigen, suchen und löschen
- Angebote anlegen, bearbeiten, anzeigen, filtern und löschen
- Positionslisten für Angebote
- Steuersätze pro Angebotsposition
- Angebote in Rechnungen umwandeln
- Rechnungen anlegen, bearbeiten, anzeigen, filtern und löschen
- Positionslisten für Rechnungen
- Steuersätze pro Rechnungsposition
- Zahlungsdatum und Mahnstufen pro Rechnung
- Dashboard mit einfachen Kennzahlen
- CSV-Export für Rechnungen
- PDF-Export für einzelne Rechnungen
- JSON-API für Kunden, Angebote, Rechnungen und Audit-Log
- JSON-basierte Datenspeicherung
- Migrationssystem für alte Datenstände
- Audit-Log für zentrale Aktionen
- automatisierte Tests für Passwortlogik, Geldparser, Positionsberechnung, Store-Flow und PDF-Erzeugung

## Technischer Ansatz

Das Projekt verwendet überwiegend die Go Standard Library und verzichtet bewusst auf große externe Abhängigkeiten.

Verwendete Bausteine:

- `net/http` für Routing und HTTP-Handling
- `html/template` für serverseitige Templates
- `encoding/json` für Persistenz und API
- `encoding/csv` für CSV-Export
- eigene einfache PDF-Erzeugung für Rechnungen
- `crypto/hmac`, `crypto/sha256` und PBKDF2-ähnliche Ableitung für Passwort-Hashing
- einfache Session-Verwaltung über HTTP-Cookies
- JSON-Migrationen im Store

## Projektstruktur

```txt
Go_Invoice_Lite/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── admin/
│   ├── api/
│   ├── app/
│   ├── auth/
│   ├── customer/
│   ├── dashboard/
│   ├── invoice/
│   ├── model/
│   ├── pdf/
│   ├── quote/
│   ├── store/
│   └── web/
├── web/
│   ├── static/
│   │   ├── app.js
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

Danach ist die Anwendung erreichbar unter:

```txt
http://localhost:8080
```

Optional kann die Adresse über eine Umgebungsvariable geändert werden:

```bash
APP_ADDR=:9090 go run ./cmd/server
```

Der Datenpfad kann ebenfalls geändert werden:

```bash
APP_DATA=data/app.json go run ./cmd/server
```

## Standard-Login

Beim ersten Start wird ein Admin-Benutzer angelegt.

```txt
Benutzername: admin
Passwort: admin123
```

Der Standard-Zugang ist nur für lokale Entwicklung vorgesehen und sollte nach dem ersten Start geändert werden.

## Datenhaltung

Die Anwendung speichert ihre Daten lokal in einer JSON-Datei.

```txt
data/app.json
```

Die Datenstruktur enthält:

- Schema-Version
- Benutzer
- Kunden
- Angebote
- Rechnungen
- Audit-Log
- interne Zähler für IDs

## Migrationssystem

Der Store besitzt eine Schema-Version. Beim Start wird der Datenbestand geprüft und bei Bedarf migriert. Ältere Angebote und Rechnungen ohne Positionslisten werden in das neue Positionsmodell übernommen.

Aktuelle Schema-Version:

```txt
2
```

## Zentrale Bereiche

### Benutzerverwaltung

Administratoren können Benutzer anlegen, bearbeiten, aktivieren, deaktivieren und löschen. Beim Bearbeiten kann ein neues Passwort gesetzt werden. Das dient als einfacher Passwort-Reset.

Normale Benutzer können ihr eigenes Passwort über den Account-Bereich ändern.

### Kunden

Kunden bilden die Grundlage für Angebote und Rechnungen. Erfasst werden Firma, Ansprechpartner, E-Mail, Telefon, Adresse und Notizen. Die Kundenliste besitzt eine Suchfunktion.

### Angebote

Angebote besitzen eine Angebotsnummer, einen Kundenbezug, Titel, Beschreibung, Status, Gültigkeitsdatum und Positionslisten. Jede Position besitzt Menge, Einzelpreis netto und eigenen Steuersatz.

Ein Angebot kann in eine Rechnung umgewandelt werden. Die Positionsliste wird dabei übernommen.

### Rechnungen

Rechnungen besitzen eine Rechnungsnummer, einen Kundenbezug, Titel, Beschreibung, Status, Rechnungsdatum, Fälligkeitsdatum, Zahlungsdatum, Mahnstufe und Positionslisten.

Das Zahlungsdatum setzt die Rechnung automatisch auf den Status `paid`.

### Dashboard

Das Dashboard zeigt einfache Kennzahlen:

- Kunden
- Angebote
- Rechnungen
- offene Rechnungen
- bezahlte Rechnungen
- offene Beträge
- bezahlte Umsätze
- letzte Angebote
- letzte Rechnungen

### Exporte

Rechnungen können als CSV-Liste exportiert werden.

```txt
GET /invoices/export.csv
```

Einzelne Rechnungen können als einfache PDF-Datei exportiert werden.

```txt
GET /invoices/{id}/pdf
```

### Audit-Log

Das Audit-Log protokolliert zentrale Aktionen:

- Login und Logout
- Benutzer anlegen, ändern, löschen
- Passwortänderungen und Passwort-Resets
- Kunden anlegen, ändern, löschen
- Angebote anlegen, ändern, löschen
- Angebot in Rechnung umwandeln
- Rechnungen anlegen, ändern, löschen

## Web-Routen

```txt
GET  /login
POST /login
GET  /logout
GET  /account/password
POST /account/password

GET  /

GET  /customers
GET  /customers/new
POST /customers/new
GET  /customers/{id}/edit
POST /customers/{id}/edit
POST /customers/{id}/delete

GET  /quotes
GET  /quotes/new
POST /quotes/new
GET  /quotes/{id}/edit
POST /quotes/{id}/edit
POST /quotes/{id}/delete
POST /quotes/{id}/convert

GET  /invoices
GET  /invoices/new
POST /invoices/new
GET  /invoices/{id}/edit
POST /invoices/{id}/edit
POST /invoices/{id}/delete
GET  /invoices/{id}/pdf
GET  /invoices/export.csv

GET  /users
GET  /users/new
POST /users/new
GET  /users/{id}/edit
POST /users/{id}/edit
POST /users/{id}/delete

GET  /audit
```

## API-Endpunkte

Die API ist über die bestehende Session geschützt. Ein API-Aufruf muss also mit einem angemeldeten Benutzer erfolgen.

```txt
GET    /api/

GET    /api/customers
POST   /api/customers
GET    /api/customers/{id}
PUT    /api/customers/{id}
DELETE /api/customers/{id}

GET    /api/quotes
POST   /api/quotes
GET    /api/quotes/{id}
PUT    /api/quotes/{id}
DELETE /api/quotes/{id}
POST   /api/quotes/{id}/convert

GET    /api/invoices
POST   /api/invoices
GET    /api/invoices/{id}
PUT    /api/invoices/{id}
DELETE /api/invoices/{id}

GET    /api/audit
```

Beispiel für eine Rechnungsposition im JSON-Format:

```json
{
  "description": "Backend-Entwicklung",
  "quantity": 2,
  "unit_net_amount": 7500,
  "tax_rate": 19
}
```

Beträge werden intern in Cent gespeichert. `7500` entspricht also `75,00 €`.

## Sicherheitshinweise

Die Anwendung enthält grundlegende Sicherheitsmaßnahmen für ein lokales MVP:

- Passwörter werden nicht im Klartext gespeichert
- Session-Cookies werden serverseitig validiert
- deaktivierte Benutzer können sich nicht anmelden
- geschützte Seiten sind nur nach Login erreichbar
- Admin-Bereiche sind rollenbeschränkt
- Beträge werden intern als Ganzzahlen verarbeitet
- zentrale Aktionen werden im Audit-Log protokolliert

Für einen produktiven Einsatz wären weitere Maßnahmen notwendig:

- CSRF-Schutz für Formulare
- HTTPS-Konfiguration
- persistente Session-Verwaltung
- sichere Passwort-Reset-Strecke mit Token und Ablaufzeit
- Datenbank mit Transaktionen
- serverseitige Eingabevalidierung mit detaillierten Feldfehlern
- strengere rollenbasierte Rechteprüfung
- revisionssicheres Audit-Log
- Backup- und Restore-Konzept
- rechtlich geprüfte Rechnungs-PDFs

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

## Mögliche Weiterentwicklung

- SQLite- oder PostgreSQL-Persistenz
- echtes Migrationssystem für SQL-Datenbanken
- CSRF-Tokens
- API-Key- oder Bearer-Token-Authentifizierung
- PDF-Layout mit Firmenangaben, Rechnungsadresse und Fußbereich
- Zahlungsübersicht und Mahnlauf
- wiederkehrende Rechnungen
- Datei-Upload für Kundendokumente
- Importfunktionen für Kunden und Positionen
- umfangreichere Testabdeckung für Handler und API

## Lizenz

Dieses Projekt kann unter der MIT-Lizenz veröffentlicht werden. Eine passende `LICENSE`-Datei kann ergänzt werden.
