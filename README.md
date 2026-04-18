# Trace

Trace is a desktop app for managing electronic components, supplier data, imported assets, and project requirements.

The app uses a Go backend with Wails v3 and a Svelte frontend. PostgreSQL stores application data, and Trace-managed files live under the local Trace home directory.

## Requirements

- Go 1.21+
- Node.js 18+
- PostgreSQL
- Wails CLI (`wails3`)

## Quick Start

Default database URL:

```bash
postgres://meet:changeme@localhost:5432/trace?sslmode=disable
```

Example PostgreSQL setup on Ubuntu:

```bash
sudo systemctl start postgresql
sudo -u postgres createuser -P meet
createdb -U meet trace
```

Install frontend dependencies:

```bash
cd frontend && npm install
```

Start the app in development mode:

```bash
DATABASE_URL=postgres://meet:changeme@localhost:5432/trace?sslmode=disable wails3 dev
```

Build a production app:

```bash
wails3 build
```

Run tests:

```bash
go test ./...
```

## Optional Provider Configuration

Trace will skip unconfigured providers. Optional environment variables:

- `DIGIKEY_CLIENT_ID`, `DIGIKEY_CLIENT_SECRET`
- `DIGIKEY_CUSTOMER_ID`, `DIGIKEY_SITE`, `DIGIKEY_LANGUAGE`, `DIGIKEY_CURRENCY`
- `MOUSER_API_KEY`
- `LCSC_ENABLED`, `LCSC_CURRENCY`

## Repository Layout

- `main.go`: Wails entry point and dependency wiring.
- `frontend/`: Svelte 5 + TypeScript + Vite UI.
- `internal/app/`: Wails bindings and frontend-facing DTOs.
- `internal/service/`: Core application workflows.
- `internal/domain/`: Domain models and repository interfaces.
- `internal/store/postgres/`: PostgreSQL persistence.
- `internal/ingest/`: File and archive ingestion pipeline.
- `internal/kicad/`: KiCad import/export and project handling.
- `cmd/seed/`: Local seed utility.
- `Taskfile.yml`: Convenience tasks for frontend install/build and Linux builds.
- `build/config.yml`: Wails dev-mode configuration.

## Notes For Contributors

- `go.mod` uses `github.com/C-Ma-P/go-easyeda` `v1.0.0` and replaces Wails with the private fork at `github.com/C-Ma-P/wails/v3`.
- CI needs a `PRIVATE_MODULES_TOKEN` secret with read access to `github.com/C-Ma-P/wails` so GitHub Actions can fetch the private module.
- Scratch notes and backlog items live in `docs/` instead of the repository root.
