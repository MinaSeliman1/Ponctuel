# Jalon 1 - ingestion et affichage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Livrer une première version reproductible de Ponctuel qui importe le GTFS statique, ingère des feeds GTFS-Realtime en mode fixture ou STM, persiste les positions/prédictions de façon idempotente et affiche une carte minimale.

**Architecture:** Un monorepo Go produit un binaire `ingester` et un binaire `api`, avec un domaine de normalisation indépendant du réseau et de PostgreSQL. TimescaleDB tourne localement via Docker Compose; le frontend Vue 3/Vite consomme une API GraphQL en lecture et peut fonctionner sur des données fixture. Redpanda, le matcher complet et le modèle ML appartiennent à des plans ultérieurs.

**Tech Stack:** Go 1.27.1, MobilityData GTFS-Realtime bindings, `pgx/v5`, TimescaleDB 2.30.0 sur PostgreSQL 17, GraphQL avec gqlgen, Vue 3 + Vite + TypeScript, Leaflet, Vitest, Playwright, Docker Compose et GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-11-ponctuel-design.md`

## Global Constraints

- Budget logiciel/hébergement : zéro dollar; aucune dépendance cloud payante.
- Le dépôt doit fonctionner en mode `fixture` sans clé API ni compte externe.
- `STM_API_KEY` reste hors du dépôt, des logs et des fixtures.
- La boucle STM démarre à 30 secondes et respecte les erreurs 401/403/429/5xx.
- La réponse STM est traitée comme protobuf GTFS-Realtime, jamais comme JSON supposé.
- Les données STM affichées comportent une attribution et aucun logo STM non autorisé.
- Les délais utilisent la convention GTFS : négatif = en avance.
- Les requêtes readiness vérifient réellement la connexion à la base.
- Chaque tâche finit par un test ciblé et un commit indépendant.

## Structure cible après le plan

```text
go.mod
go.sum
cmd/
├── api/main.go
├── fixturegen/main.go
└── ingester/main.go
services/
├── api/
│   ├── graph/schema.graphqls
│   ├── graph/generated.go
│   ├── graph/model/models_gen.go
│   ├── graph/resolver.go
│   └── http.go
└── ingester/
    ├── config/config.go
    ├── domain/model.go
    ├── domain/normalize.go
    ├── fetch/gtfsrt.go
    ├── fetch/gtfsrt_test.go
    ├── store/postgres.go
    ├── store/postgres_test.go
    └── runner/runner.go
db/migrations/
├── 001_extensions.sql
├── 002_static_gtfs.sql
├── 003_realtime.sql
└── 004_views.sql
testdata/
├── gtfs/*.txt
└── realtime/*.pb
web/
├── package.json
├── vite.config.ts
├── src/
│   ├── App.vue
│   ├── api/client.ts
│   ├── components/StatusPanel.vue
│   ├── components/VehicleMap.vue
│   └── components/VehicleTable.vue
└── tests/
deploy/compose/docker-compose.yml
.env.example
.gitignore
.github/workflows/ci.yml
README.md
```

---

### Task 1: Initialiser le dépôt et les contrats de configuration

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `.env.example`
- Create: `README.md`
- Create: `scripts/check.ps1`
- Test: `scripts/check.ps1` via PowerShell and `go test ./...` in a Go 1.27.1 container

**Interfaces:**
- Produces the module path `ponctuel`, the environment contract and the commands used by every later task.
- Consumes no application code.

- [x] **Step 1: Write the failing repository smoke check**

Create `scripts/check.ps1` with strict mode and checks for the foundation files, then invoke commands that are expected to fail because the module has no packages yet:

```powershell
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$required = @('go.mod', '.env.example', 'README.md')
foreach ($path in $required) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Fichier requis absent: $path"
    }
}

docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./...
```

Run: `pwsh -NoProfile -File scripts/check.ps1`  
Expected: FAIL because the required files and Go module are absent.

- [x] **Step 2: Add the minimal repository contract**

Create `go.mod` with `module ponctuel` and `go 1.27.1`. Create `.gitignore` that excludes `.env`, `data/`, `dist/`, `node_modules/`, coverage output, Docker volumes and generated local payloads. Create `.env.example` with non-secret values:

```dotenv
APP_ENV=fixture
INGESTER_HTTP_ADDR=:8081
API_HTTP_ADDR=:8080
DATABASE_URL=postgres://ponctuel:ponctuel@localhost:5432/ponctuel?sslmode=disable
STM_TRIP_UPDATES_URL=https://api.stm.info/pub/od/gtfs-rt/ic/v2/tripUpdates
STM_VEHICLE_POSITIONS_URL=https://api.stm.info/pub/od/gtfs-rt/ic/v2/vehiclePositions
STM_API_KEY=
STM_API_KEY_HEADER=apikey
POLL_INTERVAL=30s
RAW_DATA_DIR=./data/raw
```

Create a French README with project purpose, fixture-first quickstart, secret policy, current scope (jalon 1), source attribution and an explicit statement that no live public deployment is claimed yet.

- [x] **Step 3: Run the repository smoke check to make it pass**

Run: `pwsh -NoProfile -File scripts/check.ps1`  
Expected: PASS for file checks; `go test ./...` passes with no packages.

- [x] **Step 4: Commit the repository foundation**

```powershell
git add go.mod .gitignore .env.example README.md scripts/check.ps1
git commit -m "chore: initialize Ponctuel repository"
```

### Task 2: Construire le domaine et le parseur GTFS-Realtime

**Files:**
- Create: `services/ingester/domain/model.go`
- Create: `services/ingester/domain/normalize.go`
- Create: `services/ingester/domain/normalize_test.go`
- Create: `services/ingester/fetch/gtfsrt.go`
- Create: `services/ingester/fetch/gtfsrt_test.go`
- Create: `cmd/fixturegen/main.go`
- Create: `testdata/realtime/vehicle_positions.pb`
- Create: `testdata/realtime/trip_updates.pb`

**Interfaces:**
- `fetch.Fetcher.Fetch(ctx context.Context, feedType domain.FeedType) (domain.FeedSnapshot, error)`
- `fetch.Decode([]byte, time.Time, domain.FeedType) (domain.FeedSnapshot, error)`
- `domain.Normalize(snapshot domain.FeedSnapshot) ([]domain.Event, error)`
- `domain.FeedSnapshot` contains `FeedType`, `RecordedAt`, `SourceTimestamp`, `PayloadHash` and decoded `Entities`.
- `domain.Vehicle` contains `VehicleID`, `RouteID`, `TripID`, `Latitude`, `Longitude`, `RecordedAt` and optional `DelaySeconds`.
- `domain.Event` contains `SnapshotHash`, `FeedType`, `EntityID`, `RecordedAt`, `VehicleID`, `TripID`, `RouteID`, `StopID`, `StopSequence`, `Latitude`, `Longitude`, `PredictedAt`, `DelaySeconds`.

- [x] **Step 1: Write failing domain tests**

Add table-driven tests that assert: a vehicle position preserves latitude/longitude and timestamps; a trip update preserves negative delay; an entity without the relevant message is ignored; an absent position does not create a map point; and identical payload bytes produce the same SHA-256 hash.

Example assertion:

```go
func TestNormalizeTripUpdatePreservesNegativeDelay(t *testing.T) {
    snapshot := fixtureSnapshotWithTripUpdate(t, -90)
    events, err := Normalize(snapshot)
    if err != nil { t.Fatal(err) }
    if got := events[0].DelaySeconds; got != -90 {
        t.Fatalf("delay = %d, want -90", got)
    }
}
```

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/ingester/domain ./services/ingester/fetch -run TestNormalize -v`  
Expected: FAIL because the domain types and decoder do not exist.

- [x] **Step 2: Implement the pure domain types and normalizer**

Define typed enums for `FeedType` and `EventKind`, use `time.Time` for all internal timestamps, and make `Normalize` reject invalid coordinates, missing IDs and timestamps in a deterministic way. Keep the normalizer independent from HTTP and PostgreSQL.

Add the MobilityData dependency and decode protobuf using `google.golang.org/protobuf/proto` and `github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs`. Do not log raw payloads or headers.

- [x] **Step 3: Implement HTTP fetching with fixture mode**

Implement a client with an injected `http.Client`, base request headers from `STM_API_KEY_HEADER`, timeout, response-size limit, status classification and SHA-256 hashing. In fixture mode, read the committed `.pb` file through the same `Decode` path used by HTTP.

The fetcher must return typed errors for unauthorized, rate-limited, upstream and malformed responses. It must never include the key value in an error string.

- [x] **Step 4: Generate deterministic protobuf fixtures and rerun tests**

Implement `cmd/fixturegen` to marshal one vehicle-position feed and one trip-update feed with fixed timestamps and IDs, then run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go run ./cmd/fixturegen
docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/ingester/domain ./services/ingester/fetch -v
```

Expected: PASS, with fixture files reproducible byte-for-byte after regeneration.

- [x] **Step 5: Commit the domain and parser**

```powershell
git add services/ingester/domain services/ingester/fetch cmd/fixturegen testdata/realtime go.mod go.sum
git commit -m "feat: decode and normalize GTFS realtime feeds"
```

### Task 3: Ajouter le schéma TimescaleDB et le repository idempotent

**Files:**
- Create: `db/migrations/001_extensions.sql`
- Create: `db/migrations/002_static_gtfs.sql`
- Create: `db/migrations/003_realtime.sql`
- Create: `db/migrations/004_views.sql`
- Create: `services/ingester/store/postgres.go`
- Create: `services/ingester/store/postgres_test.go`

**Interfaces:**
- `store.Repository.InsertSnapshot(ctx context.Context, snapshot domain.FeedSnapshot) (snapshotID int64, inserted bool, err error)` returns `inserted=false` for a known payload hash.
- `store.Repository.InsertEvents(ctx context.Context, snapshotID int64, events []domain.Event) (int, error)` inserts normalized events in one transaction.
- `store.Repository.CountEvents(ctx context.Context) (int64, error)` returns the public counter.
- `store.Repository.LatestVehicles(ctx context.Context) ([]domain.Vehicle, error)` returns positions newer than the configured freshness window.

- [x] **Step 1: Write failing migration and repository tests**

Add a repository contract test that inserts the same snapshot twice and asserts one new snapshot, inserts a negative delay and asserts it is preserved, and verifies the latest-vehicle query excludes stale records. The test must start against `timescale/timescaledb:2.30.0-pg17` from Docker Compose and apply migrations before assertions.

Run: `docker compose -f deploy/compose/docker-compose.yml up -d db` followed by the repository test command.  
Expected: FAIL because Compose, migrations and repository do not exist.

- [x] **Step 2: Write migrations with explicit constraints**

Create the Timescale extension, static GTFS tables, `feed_snapshot`, `vehicle_position`, `prediction`, `arrival_observed` and `prediction_error`. Use `service_date` in trip-instance keys so the same `trip_id` on different days cannot collide. Add indexes on `(route_id, recorded_at)`, `(trip_id, service_date, stop_id)` and fresh-position queries.

Make `vehicle_position` and `prediction` hypertables on their timestamp columns. Add a conservative local retention policy for raw snapshots only; leave derived analytical rows until a later data-policy decision.

- [x] **Step 3: Implement the pgx repository**

Use `pgxpool.Pool`, parameterized SQL and explicit transactions. `InsertSnapshot` must use `ON CONFLICT (feed_type, payload_hash) DO NOTHING RETURNING snapshot_id`; when no row is returned, report `inserted=false` so the runner skips event insertion. `InsertEvents` uses the returned ID for a new snapshot. Map database failures to errors that retain operation context without query parameters.

- [x] **Step 4: Run the repository test to verify it passes**

Run: `docker compose -f deploy/compose/docker-compose.yml up -d db` and then `docker run --rm --network host -v "${PWD}:/src" -w /src -e DATABASE_URL=postgres://ponctuel:ponctuel@host.docker.internal:5432/ponctuel?sslmode=disable golang:1.27.1 go test ./services/ingester/store -v`  
Expected: PASS, including idempotency, negative delay and freshness behavior.

- [x] **Step 5: Commit persistence**

```powershell
git add db/migrations services/ingester/store
git commit -m "feat: persist GTFS realtime events in TimescaleDB"
```

### Task 4: Assembler l'ingester, la configuration et les endpoints de santé

**Files:**
- Create: `services/ingester/config/config.go`
- Create: `services/ingester/config/config_test.go`
- Create: `services/ingester/runner/runner.go`
- Create: `services/ingester/runner/runner_test.go`
- Create: `services/ingester/http.go`
- Create: `cmd/ingester/main.go`

**Interfaces:**
- `config.Load() (config.Config, error)` rejects missing STM credentials only when `APP_ENV=stm`.
- `runner.Run(ctx context.Context, cfg config.Config, source fetch.Source, repo store.Repository) error` runs one immediate collection then the configured ticker.
- `GET /healthz` returns 200 while the process is alive.
- `GET /readyz` returns 200 only when database ping succeeds and config is valid.
- `GET /metrics` exposes Prometheus text format.

- [x] **Step 1: Write failing configuration and runner tests**

Test default 30-second polling, fixture mode without a key, STM mode failing without a key, API key redaction in errors, immediate first collection, cancellation of the ticker and no duplicate inserts when a fetch returns the same hash twice.

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/ingester/config ./services/ingester/runner -v`  
Expected: FAIL because config and runner do not exist.

- [x] **Step 2: Implement configuration and one collection cycle**

Use typed `time.Duration`, URL parsing and explicit mode validation. The runner sequence is `fetch -> decode -> normalize -> insert snapshot -> insert events -> observe metrics`. A duplicate snapshot increments a duplicate counter and skips event insertion entirely.

- [x] **Step 3: Add ticker, graceful shutdown and HTTP health**

Use `context.WithCancel`, `signal.NotifyContext`, `time.NewTicker`, and a bounded database ping timeout. `/readyz` must not report ready when the database is down. Metrics must include counters for fetches, parse errors, duplicates, inserted events and upstream status classes.

- [x] **Step 4: Run service tests and binary build**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/ingester/... ./cmd/ingester/...
docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go build -o /tmp/ingester ./cmd/ingester
```

Expected: PASS and a successful Linux binary build.

- [x] **Step 5: Commit the ingester**

```powershell
git add services/ingester cmd/ingester
git commit -m "feat: run fixture and STM ingestion loop"
```

### Task 5: Importer le GTFS statique avec une source traçable

**Files:**
- Create: `cmd/gtfsimport/main.go`
- Create: `services/ingester/gtfs/static.go`
- Create: `services/ingester/gtfs/static_test.go`
- Create: `scripts/download-gtfs.ps1`
- Create: `docs/data-sources.md`
- Modify: `README.md`
- Create: `testdata/gtfs/agency.txt`
- Create: `testdata/gtfs/routes.txt`
- Create: `testdata/gtfs/trips.txt`
- Create: `testdata/gtfs/stops.txt`
- Create: `testdata/gtfs/stop_times.txt`

**Interfaces:**
- `gtfs.Import(ctx context.Context, db *pgxpool.Pool, archive io.Reader, feedVersion string) error`
- `gtfs.ValidateRequiredFiles(zipReader *zip.Reader) error`

- [x] **Step 1: Write failing importer tests**

Test that an archive containing the five required files loads rows, malformed CSV returns a useful error with file name and row number, and a second import of the same `feed_version` is rejected or replaced according to the documented policy. Use an in-memory ZIP fixture built by the test; no network.

Run: `docker run --rm --network host -v "${PWD}:/src" -w /src -e DATABASE_URL=postgres://ponctuel:ponctuel@host.docker.internal:5432/ponctuel?sslmode=disable golang:1.27.1 go test ./services/ingester/gtfs -v`  
Expected: FAIL because the importer does not exist.

- [x] **Step 2: Implement deterministic static GTFS import**

Parse CSV with `encoding/csv`, validate headers, stage rows in temporary tables and insert into the versioned static tables in a transaction. Preserve GTFS identifiers as strings. Convert `arrival_time` values beyond 24:00:00 using the GTFS service-day convention instead of rejecting them.

- [x] **Step 3: Add the download script and attribution document**

`download-gtfs.ps1` accepts a URL, destination and expected SHA-256, writes to `data/` only, and fails on checksum mismatch. `docs/data-sources.md` records the STM URL, date of retrieval, attribution, license and the fact that the endpoint can change.

- [x] **Step 4: Run the importer test and commit**

Run the importer test, then:

```powershell
git add cmd/gtfsimport services/ingester/gtfs scripts/download-gtfs.ps1 docs/data-sources.md testdata/gtfs README.md
git commit -m "feat: import versioned static GTFS schedule"
```

### Task 6: Exposer l'API GraphQL minimale

**Files:**
- Create: `services/api/graph/schema.graphqls`
- Create: `services/api/graph/resolver.go`
- Generate: `services/api/graph/generated.go`
- Generate: `services/api/graph/model/models_gen.go`
- Create: `services/api/http.go`
- Create: `cmd/api/main.go`
- Create: `services/api/http_test.go`
- Create: `tools.go`

**Interfaces:**
- GraphQL query `dashboard: Dashboard!` returns `eventCount`, `lastCollectedAt`, `mode` and `stale`.
- GraphQL query `vehicles(limit: Int = 500): [Vehicle!]!` returns `vehicleId`, `routeId`, `tripId`, `latitude`, `longitude`, `recordedAt`.
- `GET /healthz`, `GET /readyz`, `GET /metrics` have the same semantics as the ingester.
- `POST /query` is the GraphQL endpoint; introspection is allowed locally and can be disabled for a public deployment.

- [x] **Step 1: Write failing HTTP and schema tests**

Use a fake repository to assert the dashboard response, the vehicle limit, stale-data flag and readiness failure when `Ping` fails. Assert that API keys are not accepted as query parameters or logged.

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/api -v`  
Expected: FAIL because the schema and handlers do not exist.

- [x] **Step 2: Add gqlgen schema and typed resolvers**

Define the schema with explicit nullable/non-nullable fields, generate code with a pinned gqlgen version, and implement resolvers over the repository interface. The API remains read-only; it never fetches STM directly.

- [x] **Step 3: Add health, metrics and bounded GraphQL execution**

Configure request body size, HTTP timeouts, CORS from an allowlist and a query-depth limit. Return structured errors without SQL details. Add Prometheus counters for HTTP requests and GraphQL errors.

- [x] **Step 4: Run API tests and commit**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/api ./cmd/api -v`  
Expected: PASS.

```powershell
git add services/api cmd/api tools.go go.mod go.sum
git commit -m "feat: expose the live dashboard GraphQL API"
```

### Task 7: Construire l'interface Vue et la carte fixture-first

**Files:**
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/vite.config.ts`
- Create: `web/index.html`
- Create: `web/src/main.ts`
- Create: `web/src/App.vue`
- Create: `web/src/api/client.ts`
- Create: `web/src/types.ts`
- Create: `web/src/components/StatusPanel.vue`
- Create: `web/src/components/VehicleMap.vue`
- Create: `web/src/components/VehicleTable.vue`
- Create: `web/src/styles.css`
- Create: `web/tests/components/StatusPanel.test.ts`
- Create: `web/tests/components/VehicleTable.test.ts`

**Interfaces:**
- `Dashboard` and `Vehicle` TypeScript types mirror the GraphQL schema.
- `fetchDashboard(signal?: AbortSignal): Promise<Dashboard>` and `fetchVehicles(limit: number, signal?: AbortSignal): Promise<Vehicle[]>` are the only data access functions used by components.
- Components render loading, empty, stale, error and success states without hardcoded live data.

- [x] **Step 1: Write failing component tests**

Add Vitest tests for: loading text, error message without technical secrets, empty-state copy, stale-data warning, and a vehicle row showing line/trip and delay status.

Run: `pnpm --dir web test -- --run`  
Expected: FAIL because the Vue app and package scripts do not exist.

- [x] **Step 2: Scaffold Vue 3/Vite/TypeScript and API client**

Use a pinned package lock, scripts `dev`, `build`, `typecheck`, `test` and `test:e2e`. Keep the API base URL in `VITE_API_URL`, defaulting to `/query` for same-origin deployment. Do not expose `STM_API_KEY` to Vite.

- [x] **Step 3: Implement accessible dashboard components**

Use semantic headings, keyboard-accessible filters, visible focus states and French labels. Render the last collection, event count and mode. Show a clear “données vieillissantes” state when freshness exceeds the configured threshold.

- [x] **Step 4: Add deterministic SVG map without paid tile dependency**

The frontend uses an inline, accessible Montréal schematic with valid-coordinate markers and no external tile dependency. This keeps the public demo deterministic and free to run while leaving the API coordinates ready for a production map provider.

Render vehicle markers only for valid coordinates, use a configurable tile URL, include the tile provider attribution, and show an explicit map fallback when tiles fail. Do not use a STM logo or branded visual asset.

- [x] **Step 5: Run web tests and build**

Run:

```powershell
pnpm --dir web install --frozen-lockfile
pnpm --dir web test -- --run
pnpm --dir web run typecheck
pnpm --dir web run build
```

Expected: PASS, with a production bundle in `web/dist` and no API key in the generated assets.

- [x] **Step 6: Commit the frontend**

```powershell
git add web
git commit -m "feat: add fixture-first live vehicle dashboard"
```

### Task 8: Docker Compose, intégration fixture et CI publique

**Files:**
- Create: `.dockerignore`
- Create: `.env.example`
- Create: `deploy/compose/docker-compose.yml`
- Create: `deploy/compose/README.md`
- Create: `Dockerfile.ingester`
- Create: `Dockerfile.api`
- Create: `Dockerfile.web`
- Create: `scripts/smoke.ps1`
- Create: `.github/workflows/ci.yml`
- Create: `.github/ISSUE_TEMPLATE/bug_report.yml`
- Create: `.github/ISSUE_TEMPLATE/feature_request.yml`
- Create: `.github/pull_request_template.md`
- Create: `CONTRIBUTING.md`
- Create: `SECURITY.md`
- Modify: `README.md`

**Interfaces:**
- `docker compose -f deploy/compose/docker-compose.yml up --build` starts DB, ingester, API and web in fixture mode.
- `GET http://localhost:8080/healthz` and `GET http://localhost:8080/readyz` are executable smoke checks.
- `POST http://localhost:8080/query` answers dashboard and vehicles queries.
- CI runs without `STM_API_KEY` and blocks secrets from being committed.

- [x] **Step 1: Write the failing Compose smoke test**

Create a PowerShell smoke script that starts Compose, waits for `/readyz`, queries GraphQL and asserts `eventCount > 0` after fixture ingestion. Run it before implementing services in Compose.

Run: `pwsh -NoProfile -File scripts/smoke.ps1`  
Expected: FAIL because Compose and the smoke script do not exist.

- [x] **Step 2: Add pinned Compose services**

Use `timescale/timescaledb:2.30.0-pg17`, a Go 1.27.1 multi-stage image for services and a Node 24/26 image only for the web build. Mount migrations and a local raw-data volume. Provide healthchecks for DB and API. Keep the default profile fixture-only; STM mode is enabled only through a local `.env`.

- [x] **Step 3: Make the full fixture smoke test pass**

Start Compose, wait for DB readiness, run migrations, wait for API readiness, query the dashboard and vehicles, then run `docker compose down --volumes` from the named project only. Assert that the output contains no `STM_API_KEY` value and that the event count is non-zero.

- [x] **Step 4: Add public-repository CI**

Create separate jobs for Go tests, database integration, frontend tests/build, Docker build and secret scanning. Set `permissions: contents: read`, do not run STM collection in pull requests, and use only pinned action major versions. Public GitHub Actions runners are free, but no workflow may rely on a paid service.

- [x] **Step 5: Add contribution and security documentation**

Document local setup, fixture mode, how to configure `STM_API_KEY` locally, how to report a leaked secret, STM attribution, and the current boundary between demonstration readiness and production readiness. Add issue and pull-request templates with test evidence fields.

- [x] **Step 6: Run the complete verification matrix**

Run:

```powershell
pwsh -NoProfile -File scripts/check.ps1
pwsh -NoProfile -File scripts/smoke.ps1
docker compose -f deploy/compose/docker-compose.yml config
npm --prefix web run build
```

Expected: all checks pass, the API returns fixture data, the web bundle builds, Compose configuration is valid and the repository contains no secrets.

- [x] **Step 7: Commit the jalon 1 release candidate**

```powershell
git add deploy Dockerfile.ingester Dockerfile.api Dockerfile.web .github CONTRIBUTING.md SECURITY.md README.md scripts
git commit -m "feat: deliver jalon 1 local demo"
```

## Verification checklist before declaring Jalon 1 complete

- [x] `docker compose ... up --build` works from a clean checkout.
- [x] Fixture mode works with no `STM_API_KEY` and no network access to STM.
- [x] A repeated fixture payload produces one snapshot and no duplicate events.
- [x] Automated STM fetcher tests verify endpoint classification, configured header and protobuf handling without logging the key; a live STM call is intentionally not run in CI.
- [x] Static GTFS import is transactional and source-attributed.
- [x] `/readyz` fails when the database is unavailable.
- [x] GraphQL returns a non-zero event count and vehicle positions.
- [x] The web UI renders loading, error, empty, stale and success states.
- [x] `go test`, frontend tests, typecheck, frontend build, Compose config and smoke tests pass.
- [x] GitHub Actions runs the same fixture-first checks without repository secrets.
- [x] README states exactly what is live, what is fixture and what is not deployed.

## Follow-up plans

After this plan is complete and verified, create separate plans for:

1. Jalon 2 : Redpanda, matcher, arrivée `last_update`/`geofence` and Prometheus.
2. Jalon 3 : images publiées, chart Helm, k3s and GitHub Actions deployment.
3. Jalon 4 : agrégations, validation temporelle, predictor and `docs/resultats.md`.
