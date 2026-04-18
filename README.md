<p align="center">
	<img src="docs/banner.svg" alt="Trace banner with a stylized electronics workbench and circuit-board motif" width="100%" />
</p>

<h1 align="center">Trace</h1>

<p align="center">
	<strong>Desktop component inventory, imported assets, supplier sourcing, and project requirements.</strong>
</p>

<p align="center">
	Trace is a Go-powered desktop workbench for keeping parts, project constraints, KiCad-adjacent data, and sourcing decisions in one place instead of scattered across folders, notes, and increasingly suspicious spreadsheets.
</p>

<p align="center">
	<img alt="Go version" src="https://img.shields.io/badge/go-1.26.1-00ADD8?logo=go&amp;logoColor=white" />
	<img alt="Wails" src="https://img.shields.io/badge/desktop-Wails%20v3-111827?logo=go&amp;logoColor=white" />
	<img alt="Svelte" src="https://img.shields.io/badge/frontend-Svelte%205-FF3E00?logo=svelte&amp;logoColor=white" />
	<img alt="PostgreSQL" src="https://img.shields.io/badge/database-PostgreSQL-4169E1?logo=postgresql&amp;logoColor=white" />
	<img alt="Status" src="https://img.shields.io/badge/status-playful%20workbench-0f766e" />
	<a href="https://github.com/C-Ma-P/trace/actions/workflows/tests.yml">
		<img alt="Tests workflow" src="https://img.shields.io/github/actions/workflow/status/C-Ma-P/trace/tests.yml?branch=main&amp;label=tests" />
	</a>
</p>

<p align="center">
	<a href="#why-trace">Why Trace</a>
	·
	<a href="#features">Features</a>
	·
	<a href="#quick-start">Quick Start</a>
	·
	<a href="#repo-layout">Repo Layout</a>
	·
	<a href="#development">Development</a>
</p>

<p align="center">
	<sub>Engineered with questionable enthusiasm for BOMs, bins, and making local-first tooling look a little too composed.</sub>
</p>

<table>
	<tr>
		<td valign="top" width="50%">
			<strong>Purpose</strong><br />
			Manage electronic components, imported assets, supplier data, and project requirements in a desktop app backed by Go, Wails, Svelte, and PostgreSQL.
			<br /><br />
			<strong>Why it exists</strong><br />
			Because parts data is easier to trust when it lives in a real system instead of six browser tabs, a KiCad project folder, and a note titled &quot;final final parts list&quot;.
		</td>
		<td valign="top" width="50%">
			<strong>Status</strong><br />
			Active workbench. Frontend builds and Go tests run in GitHub Actions. Optional provider integrations are intentionally feature-gated by configuration.
			<br /><br />
			<strong>Fun facts</strong><br />
			Trace keeps local state under <code>~/.trace</code>, understands KiCad import/export workflows, and has phone intake support because apparently desktop apps can have side quests.
		</td>
	</tr>
</table>

## Why Trace

Trace is aimed at the messy middle of electronics work: keeping a component catalog, pairing it with real assets, matching projects against what is already on hand, and deciding what still needs to be sourced. The repo structure shows a clear split between domain logic, desktop bindings, supplier integrations, KiCad workflows, and a Svelte frontend.

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
- Wails CLI as <code>wails3</code>.
- Optional: <code>task</code> for Taskfile shortcuts.

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

Trace skips unconfigured providers rather than pretending everything is wired up. These environment variables are optional:

- <code>DIGIKEY_CLIENT_ID</code>, <code>DIGIKEY_CLIENT_SECRET</code>
- <code>DIGIKEY_CUSTOMER_ID</code>, <code>DIGIKEY_SITE</code>, <code>DIGIKEY_LANGUAGE</code>, <code>DIGIKEY_CURRENCY</code>
- <code>MOUSER_API_KEY</code>
- <code>LCSC_ENABLED</code>, <code>LCSC_CURRENCY</code>

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
	<summary><strong>Architecture at a glance</strong></summary>

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

- The frontend workflow in GitHub Actions runs <code>npm ci</code> and <code>npm run build</code> from <code>frontend/</code>.
- The Go workflow uses <code>go-version-file: go.mod</code> and runs <code>go test ./...</code>.
- The repo replaces <code>github.com/wailsapp/wails/v3</code> with the private fork <code>github.com/C-Ma-P/wails/v3</code>.
- GitHub Actions therefore needs a <code>PRIVATE_MODULES_TOKEN</code> secret to fetch private modules successfully.

## Roadmap-ish

Not a formal promise, just the obvious frontier from the codebase as it exists today:

- keep tightening KiCad import and export ergonomics,
- keep expanding sourcing and asset-enrichment workflows,
- keep polishing project planning and inventory intake flows.

## Contributing

Small, focused changes fit this repo best. If you touch the boundary between backend services and the desktop frontend, keep the Wails-facing app bindings and frontend expectations in sync.

## License

No license file is currently checked into this repository. If you plan to distribute it more broadly, this is one of the first things worth making explicit.
