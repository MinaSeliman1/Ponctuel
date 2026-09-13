# Jalon 3 Kubernetes et Helm Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fournir un chart Helm versionné pour déployer localement Ponctuel sur Kubernetes/k3s avec secrets, configuration, probes et services minimaux.

**Architecture:** Le chart déploie séparément l’API, l’ingester, le matcher et le web, ainsi qu’un StatefulSet TimescaleDB et un broker Redpanda de démonstration. Les valeurs non sensibles viennent d’un ConfigMap; les mots de passe et la clé STM viennent d’un Secret; chaque service possède une sonde de démarrage, de vivacité ou de disponibilité adaptée.

**Tech Stack:** Helm 3, Kubernetes apps/v1, StatefulSet, ConfigMap, Secret, Services ClusterIP, TimescaleDB, Redpanda.

**Spec:** `docs/superpowers/specs/2026-09-11-ponctuel-design.md`, sections 3, 8, 9, 10 et 11.

## Global Constraints

- Le chart est une cible locale/k3s; aucun hébergement cloud ni SLA public n’est annoncé.
- Les images applicatives utilisent un tag versionné configurable et `imagePullPolicy: IfNotPresent` par défaut.
- `STM_API_KEY` n’est jamais placé dans un ConfigMap, un Dockerfile, le frontend ou le dépôt.
- Les secrets applicatifs sont injectés par Secret Kubernetes; les valeurs par défaut ne sont que des identifiants de développement local.
- Les probes séparent startup, readiness et liveness pour éviter de redémarrer un service pendant ses migrations.
- La branche de travail est `jalon-3`; `main`, `jalon-1` et `jalon-2` restent inchangées.

---

### Task 1: Chart Helm, valeurs et contrats de sécurité

**Files:**
- Create: `deploy/k8s/chart/ponctuel/Chart.yaml`
- Create: `deploy/k8s/chart/ponctuel/values.yaml`
- Create: `deploy/k8s/chart/ponctuel/values.schema.json`
- Create: `deploy/k8s/chart/ponctuel/templates/_helpers.tpl`
- Create: `deploy/k8s/chart/ponctuel/templates/configmap.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/secret.yaml`

- [x] **Step 1: Write the chart metadata and schema**

Déclarer `apiVersion: v2`, `name: ponctuel`, `version: 0.3.0`, `appVersion: 0.3.0`, les repositories/tags des cinq images, les ports, les valeurs de ressources, le nom du Secret, les credentials de développement local et les bornes de `geofenceRadiusMeters` dans `values.schema.json`.

- [x] **Step 2: Render the initial chart and verify the expected failure**

Run: `docker run --rm -v "${PWD}:/src" -w /src alpine/helm:3.17.3 lint deploy/k8s/chart/ponctuel`
Expected: the scaffold renders successfully; workload validation is completed after the remaining templates are added.

- [x] **Step 3: Implement helpers, ConfigMap and Secret**

Ajouter les helpers de noms/labels, le ConfigMap non sensible (`APP_ENV`, URLs STM, polling, brokers, topics, rayon), et un Secret `stringData` contenant uniquement `POSTGRES_PASSWORD`, `DATABASE_URL` et `STM_API_KEY`. La clé STM reste vide par défaut et n’est pas référencée par le web.

- [x] **Step 4: Run chart lint**

Run: `docker run --rm -v "${PWD}:/src" -w /src alpine/helm:3.17.3 lint deploy/k8s/chart/ponctuel`
Expected: PASS once metadata, schema and templates are syntactically valid.

- [x] **Step 5: Commit**

```powershell
git add deploy/k8s/chart/ponctuel
git commit -m "feat: scaffold versioned ponctuel helm chart"
```

### Task 2: StatefulSets DB/Redpanda et services internes

**Files:**
- Create: `deploy/k8s/chart/ponctuel/templates/db-statefulset.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/db-service.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/redpanda-statefulset.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/redpanda-service.yaml`

- [x] **Step 1: Add the DB StatefulSet and persistent volume claim**

Déployer TimescaleDB avec `serviceName`, `volumeClaimTemplates`, les migrations montées depuis un ConfigMap et une probe `pg_isready`; injecter le mot de passe par `secretKeyRef` et ne jamais le mettre dans le ConfigMap.

- [x] **Step 2: Add the Redpanda StatefulSet and broker service**

Déployer un broker Redpanda de démonstration avec listener interne `redpanda:9092`, stockage persistant configurable, `runAsNonRoot` quand supporté par l’image, et probes basées sur `rpk topic list`/`rpk cluster info`. Les topics seront créés par un init container idempotent.

- [x] **Step 3: Add internal Services**

Créer des Services ClusterIP stables `db` et `redpanda`; utiliser ces noms dans `DATABASE_URL` et `REDPANDA_BROKERS`. Ajouter des labels de sélection identiques aux StatefulSets.

- [x] **Step 4: Render and inspect**

Run: `docker run --rm -v "${PWD}:/src" -w /src alpine/helm:3.17.3 template ponctuel deploy/k8s/chart/ponctuel > /tmp/ponctuel.yaml`
Expected: the rendered output contains one DB StatefulSet, one Redpanda StatefulSet, two ClusterIP Services, no `STM_API_KEY` in a ConfigMap and a PVC template.

- [x] **Step 5: Commit**

```powershell
git add deploy/k8s/chart/ponctuel/templates
git commit -m "feat: add stateful database and redpanda workloads"
```

### Task 3: Deployments applicatifs, services et probes

**Files:**
- Create: `deploy/k8s/chart/ponctuel/templates/ingester-deployment.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/ingester-service.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/matcher-deployment.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/matcher-service.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/api-deployment.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/api-service.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/web-deployment.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/web-service.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/serviceaccount.yaml`

- [x] **Step 1: Add the ingester and matcher Deployments**

Injecter le ConfigMap, le Secret, `DATABASE_URL`, `REDPANDA_BROKERS`, les topics et le fichier de stops; configurer ports 8081/8082, `startupProbe`, `readinessProbe` et `livenessProbe`, avec un Service ClusterIP pour chaque endpoint interne.

- [x] **Step 2: Add the API Deployment and Service**

Injecter le mode et la base, exposer 8080, et configurer `/healthz`, `/readyz`, `/metrics` avec des probes HTTP. Le Service reste ClusterIP; l’exposition externe est une décision d’environnement, pas une hypothèse du dépôt.

- [x] **Step 3: Add the web Deployment and Service**

Exposer le port 80 du Nginx, utiliser des probes HTTP `/`, et garder le reverse proxy vers le Service `api`. Ajouter `serviceAccountName` et `automountServiceAccountToken: false` aux workloads.

- [x] **Step 4: Render with fixture overrides**

Run: `docker run --rm -v "${PWD}:/src" -w /src alpine/helm:3.17.3 template ponctuel deploy/k8s/chart/ponctuel --set images.api.tag=0.3.0 --set images.ingester.tag=0.3.0 --set images.matcher.tag=0.3.0 --set images.web.tag=0.3.0`
Expected: four Deployments, four Services applicatifs, probes présentes et aucun conteneur avec un token Kubernetes monté automatiquement.

- [x] **Step 5: Commit**

```powershell
git add deploy/k8s/chart/ponctuel/templates
git commit -m "feat: deploy ponctuel application services on kubernetes"
```

### Task 4: Documentation, validation CI et limites d’exploitation

**Files:**
- Create: `deploy/k8s/README.md`
- Create: `deploy/k8s/values.local.example.yaml`
- Create: `deploy/k8s/chart/ponctuel/templates/NOTES.txt`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `CONTRIBUTING.md`
- Modify: `scripts/check.ps1`

- [x] **Step 1: Document local k3s deployment**

Documenter `helm lint`, `helm template`, la création du namespace et du Secret, l’installation locale, `kubectl get pods`, les probes et la suppression. Préciser que les credentials par défaut sont réservés au développement et que le live STM exige un Secret local ou GitHub.

- [x] **Step 2: Add CI chart validation**

Ajouter un job Helm avec `azure/setup-helm@v4`, `helm lint`, `helm template` et une recherche qui échoue si `STM_API_KEY` apparaît dans le ConfigMap rendu. Garder les permissions GitHub à `contents: read` et ne lancer aucun déploiement externe.

- [x] **Step 3: Extend local checks**

Ajouter la vérification Helm au script PowerShell via l’image épinglée `alpine/helm:3.17.3`, puis conserver les checks Go, frontend, Compose et smoke existants.

- [x] **Step 4: Run the complete validation matrix**

Run:

```powershell
docker run --rm -v "${PWD}:/src" -w /src alpine/helm:3.17.3 lint deploy/k8s/chart/ponctuel
docker run --rm -v "${PWD}:/src" -w /src alpine/helm:3.17.3 template ponctuel deploy/k8s/chart/ponctuel
pwsh -NoProfile -File scripts/check.ps1
git diff --check
```

Expected: Helm lint/template, Go, frontend, Compose, smoke and whitespace checks pass without a live cloud deployment.

- [x] **Step 5: Commit and publish the branch**

```powershell
git add deploy/k8s .github/workflows/ci.yml README.md CONTRIBUTING.md scripts/check.ps1
git commit -m "feat: validate ponctuel kubernetes deployment chart"
git push -u origin jalon-3
```

Open a pull request toward `jalon-2`; do not merge automatically.

Vérifications exécutées : Helm lint/template, `scripts/check.ps1`, `go vet ./...`,
smoke Compose (`2 événements, 1 véhicule`) et `git diff --check`. Le script
shell Redpanda est forcé en LF par `.gitattributes` pour rester portable entre
Windows et Alpine.
