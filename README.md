# Trace

A desktop app for managing electronic component inventory and project requirements.

Built with [Wails v3](https://v3.wails.io) (Go backend) + Svelte frontend, backed by PostgreSQL.

## Setup

**Database:** PostgreSQL required. Default connection: `postgres://meet:changeme@localhost:5432/trace?sslmode=disable` (override with `DATABASE_URL` env var).

**On Ubuntu:**
```bash
sudo systemctl start postgresql
sudo -u postgres createuser -P meet  # password: changeme
createdb -U meet trace
```
(`createdb` is a PostgreSQL tool; run as your current user)

**Sourcing APIs** (optional, via env vars):
- `DIGIKEY_CLIENT_ID`, `DIGIKEY_CLIENT_SECRET`
- `DIGIKEY_CUSTOMER_ID`, `DIGIKEY_SITE`, `DIGIKEY_LANGUAGE`, `DIGIKEY_CURRENCY`
- `MOUSER_API_KEY`
- `LCSC_ENABLED`, `LCSC_CURRENCY`

Unconfigured providers are skipped.


## Development

**Requirements:** Go 1.21+, Node.js 18+, Wails CLI, PostgreSQL

**Dev mode:**
```bash
DATABASE_URL=postgres://localhost:5432/trace?sslmode=disable wails3 dev
```

**Build:**
```bash
wails3 build
```


## Architecture

```
main.go                   Wails entry point
internal/app/            Wails binding layer (frontend ↔ DTOs)
internal/domain/         Core domain models and repository interfaces
internal/domain/registry/Canonical attribute definitions per component type
internal/service/        Business logic (domain-agnostic)
internal/store/postgres/ PostgreSQL repositories
frontend/                Svelte 5 + TypeScript + Vite
```

Domain and service layers are database-agnostic. Postgres-specific code is isolated in `internal/store/postgres/`.
