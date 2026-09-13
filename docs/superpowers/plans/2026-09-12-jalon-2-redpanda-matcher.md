# Jalon 2 Redpanda et matcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Séparer le transport des événements GTFS-Realtime avec Redpanda, calculer deux observations d’arrivée déterministes et exposer des métriques Prometheus dans l’ingester et le matcher.

**Architecture:** L’ingester conserve l’écriture durable PostgreSQL du jalon 1 puis publie chaque événement normalisé dans le topic Redpanda correspondant. Le matcher consomme les deux topics avec un groupe Kafka dédié, garde les prédictions récentes en mémoire et écrit des arrivées idempotentes dans PostgreSQL selon `last_update` ou `geofence`. Le mode fixture reste le chemin par défaut et fournit un fichier de stops minimal pour la démonstration lorsque le GTFS statique n’est pas encore importé.

**Tech Stack:** Go 1.27.1, Kafka-compatible Redpanda, `segmentio/kafka-go`, PostgreSQL/TimescaleDB, Docker Compose, Prometheus text exposition, tests Go et smoke PowerShell.

**Spec:** `docs/superpowers/specs/2026-09-11-ponctuel-design.md`, sections 3, 4, 6, 7, 9 et 10.

## Global Constraints

- Aucun service payant ni secret STM n’est requis pour les tests, la CI ou le smoke Compose.
- Les topics sont nommés exactement `trip-updates` et `vehicle-positions`.
- Les événements sont encodés en JSON versionné, sans payload GTFS brut ni clé d’API.
- Les écritures d’arrivées sont idempotentes par `(trip_id, service_date, stop_id, method)`.
- `last_update` signifie qu’une prédiction publiée est échue au moment où elle est consommée.
- `geofence` utilise une distance haversine et un rayon borné par configuration, avec une confiance explicite.
- La branche de travail est `jalon-2`; `main` reste inchangée.

---

### Task 1: Contrat d’événement, configuration et schéma d’arrivée

**Files:**
- Create: `services/events/event.go`
- Create: `services/events/event_test.go`
- Create: `db/migrations/005_arrival_methods.sql`
- Modify: `services/ingester/domain/model.go`
- Modify: `services/ingester/config/config.go`
- Modify: `services/ingester/config/config_test.go`

**Interfaces:**
- `events.Message` encode les champs de `domain.Event` avec `schema_version: 1`.
- `events.TopicFor(domain.EventKind)` retourne `trip-updates` ou `vehicle-positions`.
- `config.Config` expose `RedpandaBrokers []string`, `RedpandaGroupID string` et `GeofenceRadiusMeters float64`.
- `domain.ArrivalObserved` contient `TripID`, `ServiceDate`, `StopID`, `StopSequence`, `ObservedAt`, `Method`, `Confidence` et `Reason`.

- [x] **Step 1: Write the failing tests**

Ajouter des tests qui vérifient que l’événement JSON conserve un timestamp UTC et les champs `trip_id`, `stop_id`, `predicted_at`; que les deux topics sont sélectionnés; que `REDPANDA_BROKERS` est séparé sur les virgules; que le rayon par défaut est positif; et qu’un `ArrivalObserved` refuse une méthode inconnue.

- [x] **Step 2: Run tests to verify they fail**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/events ./services/ingester/config`
Expected: FAIL because the event package, configuration fields and arrival model do not exist.

- [x] **Step 3: Implement the minimal contracts and migration**

Créer le contrat JSON avec `encoding/json`, ajouter les constantes `TopicTripUpdates` et `TopicVehiclePositions`, ajouter la configuration optionnelle et créer la migration qui ajoute `method`, `confidence`, `reason`, une valeur par défaut et l’index unique logique tout en conservant `source` pour compatibilité avec le schéma du jalon 1.

- [x] **Step 4: Run focused tests**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/events ./services/ingester/config`
Expected: PASS.

- [x] **Step 5: Commit**

```powershell
git add services/events services/ingester/domain/model.go services/ingester/config db/migrations/005_arrival_methods.sql
git commit -m "feat: define jalon 2 event and arrival contracts"
```

### Task 2: Publisher Redpanda et publication depuis l’ingester

**Files:**
- Create: `services/bus/kafka.go`
- Create: `services/bus/kafka_test.go`
- Modify: `services/ingester/runner/runner.go`
- Modify: `services/ingester/runner/runner_test.go`
- Modify: `cmd/ingester/main.go`
- Modify: `services/ingester/http.go`

**Interfaces:**
- `bus.Publisher` expose `Publish(context.Context, string, string, domain.Event) error` et `Close() error`.
- `bus.KafkaPublisher` utilise `kafka-go`, encode `events.Message` et publie la clé déterministe `snapshot_hash/entity_id/stop_id`.
- `runner.NewWithMetricsAndPublisher` accepte un publisher nil; nil conserve le comportement du jalon 1.

- [x] **Step 1: Write the failing tests**

Ajouter un fake publisher au test runner et vérifier qu’un événement normalisé est publié dans le bon topic après son écriture en repository, qu’une erreur de publication est remontée sans afficher de secret, et que le publisher nil ne bloque pas la collecte. Ajouter un test de round-trip du writer JSON avec un `kafka.Message` local.

- [x] **Step 2: Run tests to verify they fail**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/bus ./services/ingester/runner`
Expected: FAIL because `services/bus` and the publisher hook do not exist.

- [x] **Step 3: Implement the publisher and metrics**

Ajouter `kafka-go`, créer un writer par topic avec `RequiredAcks: kafka.RequireAll`, désactiver toute journalisation du payload, ajouter les compteurs `publish_total` et `publish_errors_total`, et appeler le publisher uniquement après l’insertion PostgreSQL réussie.

- [x] **Step 4: Run focused tests**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/bus ./services/ingester/runner`
Expected: PASS.

- [x] **Step 5: Commit**

```powershell
git add go.mod go.sum services/bus services/ingester/runner services/ingester/http.go cmd/ingester/main.go
git commit -m "feat: publish normalized events to redpanda"
```

### Task 3: Moteur matcher et repository des arrivées

**Files:**
- Create: `services/matcher/engine.go`
- Create: `services/matcher/engine_test.go`
- Create: `services/matcher/metrics.go`
- Modify: `services/ingester/store/postgres.go`
- Modify: `services/ingester/store/postgres_test.go`

**Interfaces:**
- `domain.Stop` contient `StopID`, `Latitude`, `Longitude`.
- `matcher.Engine.HandleEvent(domain.Event, []Stop) ([]domain.ArrivalObserved, error)` produit au plus une arrivée par méthode et arrêt.
- `store.Repository` ajoute `InsertArrival(context.Context, domain.ArrivalObserved) (bool, error)` et `LatestStops(context.Context) ([]matcher.Stop, error)`.

- [x] **Step 1: Write the failing tests**

Tester une prédiction échue qui produit une arrivée `last_update` avec confiance `0.70`; une position à moins du rayon qui produit `geofence` avec confiance décroissante selon la distance; une position hors rayon qui ne produit rien; un trip différent qui ne matche pas; un timestamp invalide qui est rejeté; et deux événements répétés qui ne produisent pas de doublon.

- [x] **Step 2: Run tests to verify they fail**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/matcher ./services/ingester/store`
Expected: FAIL because the matcher package and repository methods do not exist.

- [x] **Step 3: Implement the pure matcher rules**

Maintenir les prédictions par clé `(trip_id, service_date, stop_id)`; pour `last_update`, accepter seulement `predicted_at <= recorded_at`; pour `geofence`, calculer la distance haversine, exiger `<= radius`, un `trip_id` identique et une position valide, puis produire une confiance dans `[0,1]` et une raison explicite.

- [x] **Step 4: Implement idempotent PostgreSQL writes**

Ajouter l’insert `ON CONFLICT (trip_id, service_date, stop_id, method) DO NOTHING`, conserver `source = method`, et charger les stops du dernier `gtfs_feed_version` avec coordonnées non nulles. Retourner le nombre d’insertions via le compteur matcher.

- [x] **Step 5: Run focused tests**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/matcher ./services/ingester/store`
Expected: PASS.

- [x] **Step 6: Commit**

```powershell
git add services/matcher services/ingester/store db/migrations/005_arrival_methods.sql
git commit -m "feat: infer idempotent arrivals with two matcher methods"
```

### Task 4: Service matcher, consommation Kafka et métriques

**Files:**
- Create: `cmd/matcher/main.go`
- Create: `services/matcher/service.go`
- Create: `services/matcher/http.go`
- Create: `services/matcher/service_test.go`
- Modify: `services/ingester/config/config.go`

**Interfaces:**
- `matcher.Service` consomme les deux topics avec le groupe `ponctuel-matcher`, charge les stops au démarrage et expose `Run(context.Context) error`.
- `GET /healthz`, `GET /readyz` et `GET /metrics` sont disponibles sur `MATCHER_HTTP_ADDR`, par défaut `:8082`.
- Les métriques incluent `ponctuel_matcher_events_total`, `ponctuel_matcher_arrivals_total`, `ponctuel_matcher_duplicates_total` et `ponctuel_matcher_errors_total`.

- [x] **Step 1: Write the failing tests**

Tester les endpoints sans broker réel avec un service construit autour d’un reader fake: health retourne 200, readiness retourne 503 avant connexion DB et 200 après, et metrics contient les quatre séries avec des valeurs numériques.

- [x] **Step 2: Run tests to verify they fail**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/matcher`
Expected: FAIL because the service and handlers do not exist.

- [x] **Step 3: Implement the service**

Créer deux readers Kafka, utiliser un groupe partagé et traiter chaque message dans le contexte courant; décoder `events.Message`, appeler le moteur, insérer les arrivées et commit le message seulement après traitement. Un message invalide augmente l’erreur puis est commit pour éviter de bloquer la partition; l’erreur n’inclut ni payload ni mot de passe.

- [x] **Step 4: Run focused tests**

Run: `docker run --rm -v "${PWD}:/src" -w /src golang:1.27.1 go test ./services/matcher`
Expected: PASS.

- [x] **Step 5: Commit**

```powershell
git add services/matcher cmd/matcher services/ingester/config
git commit -m "feat: add redpanda matcher service"
```

### Task 5: Compose Redpanda, fixture stops et smoke d’intégration

**Files:**
- Create: `Dockerfile.matcher`
- Create: `deploy/compose/redpanda-init.sh`
- Modify: `deploy/compose/docker-compose.yml`
- Modify: `scripts/smoke.ps1`
- Modify: `.env.example`
- Modify: `deploy/compose/README.md`
- Modify: `README.md`
- Modify: `.github/workflows/ci.yml`
- Modify: `scripts/check.ps1`

**Interfaces:**
- Compose démarre `redpanda`, crée les topics, puis démarre ingester, matcher, API et web en fixture.
- Les variables `REDPANDA_BROKERS`, `REDPANDA_GROUP_ID`, `MATCHER_HTTP_ADDR` et `GEOFENCE_RADIUS_METERS` sont documentées.
- Le smoke attend `/readyz` du matcher et vérifie ses métriques ainsi qu’au moins une arrivée `last_update` ou `geofence`.

- [x] **Step 1: Write the failing smoke assertions**

Modifier `scripts/smoke.ps1` pour attendre `http://127.0.0.1:8082/readyz`, lire `/metrics`, exiger `ponctuel_matcher_events_total` et `ponctuel_matcher_arrivals_total`, puis lancer le smoke avant les changements Compose pour constater l’absence du service.

- [x] **Step 2: Add pinned Redpanda and matcher services**

Ajouter Redpanda avec un listener interne `redpanda:9092`, un service init qui crée exactement les deux topics, le matcher avec healthcheck, et les dépendances `service_completed_successfully`/`service_healthy`. Le mode fixture pointe vers `testdata/gtfs/stops.txt` pour les coordonnées de démonstration et le mode STM utilise les stops importés PostgreSQL.

- [x] **Step 3: Wire the ingester publisher and run the smoke**

Passer `REDPANDA_BROKERS=redpanda:9092` à l’ingester et au matcher, démarrer Compose avec un nom de projet dédié, vérifier API, web, topics et métriques, puis supprimer uniquement les ressources de ce projet.

Run: `pwsh -NoProfile -File scripts/smoke.ps1`
Expected: PASS with a non-zero event count and non-zero matcher activity.

- [x] **Step 4: Update public documentation and CI**

Documenter le rôle de Redpanda, le démarrage fixture gratuit et la limite du déploiement live; faire exécuter le smoke ou au minimum `docker compose config` et les tests Go dans GitHub Actions sans `STM_API_KEY`.

- [x] **Step 5: Commit**

```powershell
git add Dockerfile.matcher deploy/compose scripts/smoke.ps1 .env.example README.md .github/workflows/ci.yml
git commit -m "feat: integrate redpanda and matcher in compose"
```

### Task 6: Vérification complète et publication de la branche

**Files:**
- Modify: `docs/superpowers/plans/2026-09-12-jalon-2-redpanda-matcher.md`

- [x] **Step 1: Run the complete verification matrix**

Run:

```powershell
pwsh -NoProfile -File scripts/check.ps1
pwsh -NoProfile -File scripts/smoke.ps1
docker compose -f deploy/compose/docker-compose.yml config
git diff --check
```

Expected: all checks pass, no secret appears in logs or assets, and only the named smoke project is removed.

- [x] **Step 2: Mark the plan complete and inspect the diff**

Cocher les étapes réellement vérifiées, relire le diff, confirmer que `main` et `jalon-1` ne contiennent aucune modification locale, puis vérifier `git status --short --branch`.

- [x] **Step 3: Push the branch**

```powershell
git push -u origin jalon-2
```

Créer une pull request vers la branche par défaut uniquement après que les tests et le smoke soient verts; ne pas fusionner automatiquement.

Vérifications exécutées : `scripts/check.ps1`, `scripts/smoke.ps1`, `go vet ./...`,
tests PostgreSQL et GTFS séquentiels sur TimescaleDB, `docker compose config` et
`git diff --check`. La branche est poussée après le commit final; aucune fusion
automatique vers `main` n’est effectuée.
