# Trace

A desktop application for managing electronic component inventory and project requirements.

Built with [Wails v3](https://v3.wails.io) (Go backend) + Svelte frontend, backed by PostgreSQL.

---

## Runtime model

### Database

PostgreSQL is the only supported database backend.

The app connects to Postgres using a connection URL — it does not manage,
start, or locate a Postgres installation. Postgres is treated as infrastructure
that is running independently of the desktop app.

| Mode | Typical setup |
|---|---|
| Local (Ubuntu) | `postgresql` systemd service, started automatically at boot |
| Local (other) | Any locally accessible Postgres instance |
| Remote | Any network-accessible Postgres host |
| Containerized | Docker/Podman Postgres container with a published port |

The app does not care where Postgres stores its on-disk files or how it is
managed by the OS. It only needs a valid connection URL.

### Configuration

**`DATABASE_URL`** — Postgres connection URL (optional).

```
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable
```

If not set, the app defaults to:

```
postgres://localhost:5432/componentmanager?sslmode=disable
```

This default works for a standard package-managed Postgres on Ubuntu where the
`componentmanager` database has been created and the local user has access (e.g.
via `peer` auth or a password).

Supplier sourcing is configured at the app level with environment variables:

| Variable | Purpose |
|---|---|
| `DIGIKEY_CLIENT_ID` | DigiKey API client ID |
| `DIGIKEY_CLIENT_SECRET` | DigiKey API client secret |
| `DIGIKEY_CUSTOMER_ID` | Optional DigiKey customer ID |
| `DIGIKEY_SITE` | Optional DigiKey locale site, for example `US` or `DE` |
| `DIGIKEY_LANGUAGE` | Optional DigiKey locale language, for example `en` |
| `DIGIKEY_CURRENCY` | Optional DigiKey locale currency, for example `USD` |
| `MOUSER_API_KEY` | Mouser API key |
| `LCSC_ENABLED` | Optional boolean to disable LCSC sourcing |
| `LCSC_CURRENCY` | Optional LCSC currency override |

If a provider is not configured, it is skipped and the Project Plan sourcing UI
shows it as disabled instead of failing the request.

### Starting Postgres (Ubuntu)

```bash
# Check status
sudo systemctl status postgresql

# Start if not running
sudo systemctl start postgresql

# Create the database (first-time setup)
createdb componentmanager
```

### Starting the app

The desktop app is launched on demand. Postgres should already be running when
the app starts. If the database is unavailable at startup, the app will exit
with a clear error message describing the connection target and likely causes.

---

## Development

### Prerequisites

- Go 1.21+
- Node.js 18+
- Wails CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.74`
- A running PostgreSQL instance

### Run in development mode

```bash
DATABASE_URL=postgres://localhost:5432/componentmanager?sslmode=disable wails3 dev
```

### Build

```bash
wails3 build
```

---

## Architecture

```
main.go                  — Wails entry point: config, DB wiring, migrations, startup
internal/app/            — Wails binding layer (frontend ↔ domain DTOs)
internal/domain/         — Core domain models and repository interfaces
internal/domain/registry/— Canonical attribute definitions per component category
internal/service/        — Business logic, depends only on domain interfaces
internal/store/postgres/ — PostgreSQL repositories (only Postgres-specific code lives here)
frontend/                — Svelte 5 + TypeScript + Vite

Wails v3 uses Task for builds; this repo includes `Taskfile.yml` and `build/config.yml` for `wails3 dev`.
```

Postgres-specific implementation is confined to `internal/store/postgres/` and
startup wiring in `main.go`. The domain and service layers have no knowledge of
the database backend.
