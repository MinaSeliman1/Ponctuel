# Jalon 4 Mesure et predictor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Calculer les erreurs d’arrivée de façon traçable et fournir un predictor gradient boosting reproductible avec validation temporelle, sans publier de résultat inventé.

**Architecture:** PostgreSQL relie chaque arrivée aux versions de prédictions déjà observées et expose des agrégats par ligne et horizon via le repository/API. Un service Python indépendant entraîne un modèle sur des observations tabulaires, sépare le passé du futur avant l’entraînement, compare le modèle à une baseline STM et expose un contrat `/predict` uniquement quand un artefact entraîné est fourni.

**Tech Stack:** PostgreSQL/TimescaleDB, GraphQL gqlgen, Python 3.12, FastAPI, scikit-learn `GradientBoostingRegressor`, pytest.

**Spec:** `docs/superpowers/specs/2026-09-11-ponctuel-design.md`, sections 4, 6, 7, 9, 10 et 11.

## Global Constraints

- Les horizons sont calculés en secondes depuis `prediction.recorded_at` vers `prediction.predicted_at`, bornés dans `[0, 3600]`.
- La validation temporelle trie par `recorded_at` et ne mélange jamais une observation future dans l’entraînement.
- Toute métrique est accompagnée d’un nombre d’observations et d’une baseline comparable.
- Les fixtures ne sont pas présentées comme des données STM réelles et ne servent pas à annoncer une amélioration ML.
- Le predictor ne reçoit jamais `STM_API_KEY` et n’accède pas directement au broker ou à PostgreSQL dans le jalon 4.
- La branche de travail est `jalon-4`; aucune fusion automatique vers `jalon-3`.

---

### Task 1: Relier arrivées, prédictions et agrégats d’erreur

**Files:**
- Create: `db/migrations/006_prediction_error_indexes.sql`
- Modify: `services/ingester/domain/model.go`
- Modify: `services/ingester/store/postgres.go`
- Modify: `services/ingester/store/postgres_test.go`

- [x] **Step 1: Write the failing repository test**

Ajouter un test DB qui insère deux prédictions à horizons distincts, une arrivée, vérifie la création idempotente de `prediction_error`, puis vérifie que `ErrorSummary` retourne l’horizon, la moyenne, le taux à l’heure et le nombre d’observations attendus.

- [x] **Step 2: Implement the arrival/error transaction**

Dans `InsertArrival`, insérer l’arrivée puis créer les lignes `prediction_error` correspondantes avec `observed_at - predicted_at`, la fenêtre asymétrique `[-60, 300]` et `ON CONFLICT DO NOTHING`. Ajouter une vue de résumé ou une requête groupée avec horizon borné.

- [x] **Step 3: Add indexes and the domain contract**

Indexer `prediction_error` par arrivée et prediction, ajouter `domain.ErrorSummary` et `Repository.ErrorSummary(ctx, limit)`, avec une limite maximale de 500.

- [x] **Step 4: Run database-backed tests**

Run: `go test ./services/ingester/store`
Expected: PASS against the TimescaleDB service with migration 006.

- [x] **Step 5: Commit**

```powershell
git add db/migrations/006_prediction_error_indexes.sql services/ingester/domain/model.go services/ingester/store
git commit -m "feat: persist and aggregate prediction errors"
```

### Task 2: Exposer la qualité par GraphQL

**Files:**
- Modify: `services/api/graph/schema.graphqls`
- Modify: `services/api/graph/schema.resolvers.go`
- Modify: `services/api/graph/model/models_gen.go`
- Modify: `services/api/graph/generated.go`
- Modify: `services/api/http_test.go`

- [x] **Step 1: Extend the schema**

Ajouter `ErrorSummary { routeId: String, horizonSeconds: Int!, sampleCount: Int!, meanErrorSeconds: Float!, onTimeRate: Float! }` et `Query.errorSummary(limit: Int = 100): [ErrorSummary!]!`.

- [x] **Step 2: Regenerate gqlgen and implement the resolver**

Limiter la valeur à 500, rejeter les limites négatives, masquer les erreurs backend comme les autres resolvers et mapper les valeurs nullable sans exposer de SQL.

- [x] **Step 3: Update API fakes and tests**

Tester le résultat GraphQL, la limite 500 et le rejet d’une limite négative avec un fake repository retournant deux résumés.

- [x] **Step 4: Run API tests**

Run: `go test ./services/api ./services/ingester/...`
Expected: PASS.

- [x] **Step 5: Commit**

```powershell
git add services/api services/ingester
git commit -m "feat: expose prediction error summaries in graphql"
```

### Task 3: Predictor Python et validation temporelle

**Files:**
- Create: `services/predictor/__init__.py`
- Create: `services/predictor/model.py`
- Create: `services/predictor/app.py`
- Create: `services/predictor/test_model.py`
- Create: `services/predictor/test_app.py`
- Create: `services/predictor/requirements.txt`

- [x] **Step 1: Write failing temporal split and baseline tests**

Tester que les observations sont triées par `recorded_at`, que la dernière tranche est réservée au test, qu’une date future n’entre jamais dans l’entraînement, et que la baseline STM utilise directement `error_seconds` prédit par zéro additionnel.

- [x] **Step 2: Implement the feature contract**

Définir `Observation`, `temporal_split`, `feature_matrix`, `baseline_metrics` et `train_gradient_boosting`; utiliser `route_id` dans un encodage déterministe, `horizon_seconds`, heure locale et délai, avec `random_state=42`.

- [x] **Step 3: Implement FastAPI `/healthz` and `/predict`**

Charger un artefact JSON/Joblib configuré par `PREDICTOR_MODEL_PATH`; retourner 503 si aucun modèle n’est entraîné, valider les horizons `[0, 3600]` et ne jamais loguer les observations complètes.

- [x] **Step 4: Run Python tests**

Run: `python -m pytest services/predictor -q`
Expected: PASS with temporal split, baseline, model and HTTP contract tests.

- [x] **Step 5: Commit**

```powershell
git add services/predictor
git commit -m "feat: add temporally validated arrival predictor"
```

### Task 4: Packaging, documentation et CI

**Files:**
- Create: `Dockerfile.predictor`
- Create: `docs/resultats.md`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `CONTRIBUTING.md`
- Modify: `scripts/check.ps1`

- [x] **Step 1: Package the predictor**

Créer une image Python non-root, installer les dépendances verrouillées, exposer 8090 et ajouter une probe `/healthz`; le modèle reste monté par volume ou fourni par un pipeline externe.

- [x] **Step 2: Document method and honest results**

Décrire la définition de l’erreur, la fenêtre « à l’heure », la séparation temporelle, la baseline et la limite des fixtures. Ne publier aucun chiffre présenté comme STM réel sans dataset et période documentés.

- [x] **Step 3: Add CI and local checks**

Ajouter un job Python sans secret, construire l’image predictor, lancer pytest et exécuter `git diff --check`; garder le smoke fixture et les jobs précédents.

- [x] **Step 4: Run complete verification**

Run: `python -m pytest services/predictor -q`, `go test ./...`, `go vet ./...`, `scripts/check.ps1`, `scripts/smoke.ps1`, `docker compose config`, `git diff --check`.

- [x] **Step 5: Commit and publish**

```powershell
git add Dockerfile.predictor services/predictor docs/resultats.md .github/workflows/ci.yml README.md CONTRIBUTING.md scripts/check.ps1
git commit -m "feat: deliver jalon 4 measurement and predictor"
git push -u origin jalon-4
```

Open a pull request toward `jalon-3`; do not merge automatically.
