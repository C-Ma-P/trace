# Trace

[![Go version](https://img.shields.io/badge/go-1.26.1-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Desktop stack](https://img.shields.io/badge/desktop-Wails%20v3-111827)](https://v3alpha.wails.io/)
[![Tests](https://img.shields.io/github/actions/workflow/status/C-Ma-P/trace/tests.yml?branch=main&label=tests)](https://github.com/C-Ma-P/trace/actions/workflows/tests.yml)

Trace is a local-first desktop application for managing electronic components, imported assets, supplier sourcing, and project requirements. It combines a Go service layer with Wails v3, a Svelte 5 frontend, and PostgreSQL so inventory, KiCad-adjacent workflows, and sourcing decisions stay in one place.

The repository is organized around domain models, service workflows, desktop bindings, supplier integrations, KiCad tooling, and a frontend intended for day-to-day use rather than demo polish.

## Why Trace

Trace is aimed at the practical parts of electronics work: maintaining a component catalog, attaching usable assets, checking projects against available inventory, and deciding what still needs to be sourced. The codebase keeps those concerns separated across domain logic, desktop bindings, provider integrations, KiCad workflows, and the frontend.

## Features

- Component cataloging with category-specific attributes, inventory quantities, and local storage.
- Project requirement management with typed constraints and matching against on-hand parts.
- Candidate planning and saved supplier offers for project requirements.
- KiCad project discovery, preview import, and export of ready parts back into KiCad-oriented output.
- Component asset handling for symbols, footprints, datasheets, and 3D models.
- Activity reporting across app, sourcing, export, and phone intake flows.
- Optional provider integrations for DigiKey, Mouser, LCSC, and EasyEDA-related ingestion paths.
- Seed tooling for populating the local database with realistic hobbyist parts and projects.

## Typical Workflow

1. Create or import components and attach usable assets.
2. Define project requirements and constraints.
3. Match against on-hand inventory or gather supplier offers.
4. Prefer candidates, resolve gaps, and export ready parts for KiCad work.

## Quick Start

### Prerequisites

- Go 1.26.1 or newer.
- Node.js 18 or newer. The current Actions workflow uses Node.js 22.
- PostgreSQL.
- Wails CLI as `wails3`.
- Optional: `task` for Taskfile shortcuts.

### Default database URL

```bash
postgres://meet:changeme@localhost:5432/trace?sslmode=disable
```

### Example PostgreSQL setup on Ubuntu

```bash
sudo systemctl start postgresql
sudo -u postgres createuser -P meet
createdb -U meet trace
```

### Install frontend dependencies

```bash
cd frontend
npm install
```

### Run the desktop app in development mode

```bash
DATABASE_URL=postgres://meet:changeme@localhost:5432/trace?sslmode=disable wails3 dev
```

### Build locally

```bash
task linux:build DEV=true
./bin/trace
```

### Build a production app bundle

```bash
wails3 build
```

### Run tests

```bash
go test ./...
```

### Seed local sample data

```bash
go run ./cmd/seed --wipe
```

<details>
	<summary><strong>Optional provider configuration</strong></summary>

Trace skips unconfigured providers. These environment variables are optional:

- `DIGIKEY_CLIENT_ID`, `DIGIKEY_CLIENT_SECRET`
- `DIGIKEY_CUSTOMER_ID`, `DIGIKEY_SITE`, `DIGIKEY_LANGUAGE`, `DIGIKEY_CURRENCY`
- `MOUSER_API_KEY`
- `LCSC_ENABLED`, `LCSC_CURRENCY`

</details>

## Repo Layout

```text
.
|- main.go                     # Wails entry point and application wiring
|- frontend/                  # Svelte 5 + TypeScript + Vite UI
|- internal/app/              # Desktop-facing bindings and response types
|- internal/service/          # Core workflows: components, projects, sourcing, export
|- internal/domain/           # Domain models and repository interfaces
|- internal/store/postgres/   # PostgreSQL-backed persistence
|- internal/ingest/           # Asset and import ingestion pipeline
|- internal/kicad/            # KiCad import, export, and project parsing
|- internal/phoneintake/      # Phone intake server and related flows
|- internal/activity/         # Activity reporting and event plumbing
|- cmd/seed/                  # Local seed utility for dev data
|- Taskfile.yml               # Frontend install/build and Linux build shortcuts
`- build/config.yml           # Wails dev-mode configuration
```

<details>
	<summary><strong>Architecture overview</strong></summary>

```mermaid
flowchart LR
		UI[Svelte UI] --> Wails[Wails bindings]
		Wails --> Service[Go service layer]
		Service --> DB[(PostgreSQL)]
		Service --> Home[~/.trace]
		Service --> KiCad[KiCad import/export]
		Service --> Providers[Supplier and asset providers]
		Service --> Intake[Phone intake + activity streams]
```

</details>

## Development

- GitHub Actions runs `npm ci` and `npm run build` from `frontend/`.
- The Go workflow uses the version declared in `go.mod` and runs `go test ./...`.
- The repo replaces `github.com/wailsapp/wails/v3` with the private fork `github.com/C-Ma-P/wails/v3`.
- GitHub Actions therefore needs a `PRIVATE_MODULES_TOKEN` secret to fetch private modules successfully, and local builds need equivalent access to that module.

## Current Focus

These are the areas the current codebase is already investing in:

- tightening KiCad import and export workflows,
- expanding sourcing and asset-enrichment workflows,
- improving project planning and inventory intake flows.

## Contributing

Small, focused changes fit this repo best. If you touch the boundary between backend services and the desktop frontend, keep the Wails-facing app bindings and frontend expectations in sync.

## License

No license file is currently checked into this repository. If you plan to distribute it more broadly, this is one of the first things worth making explicit.
