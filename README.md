# Ponctuel

Ponctuel mesure l’écart entre les prédictions d’arrivée des autobus de la STM
et les arrivées observées. Le jalon 1 fournit une démonstration complète,
reproductible et gratuite : ingestion GTFS-Realtime, persistance TimescaleDB,
API GraphQL et dashboard Vue.

## Démonstration en une commande

Prérequis : Docker Desktop et PowerShell 7.

```powershell
Copy-Item .env.example .env
docker compose -f deploy/compose/docker-compose.yml up --build
```

Ouvre ensuite [http://localhost:3000](http://localhost:3000). L’API expose
`/healthz`, `/readyz`, `/metrics` et `POST /query` sur le port 8080.

Le mode par défaut est `fixture` : il ne contacte pas STM, ne demande aucune
clé et fonctionne sans service payant. Le smoke test reproductible démarre et
nettoie uniquement son projet Docker nommé :

```powershell
pwsh -NoProfile -File scripts/smoke.ps1
```

## Architecture

```text
GTFS-Realtime fixture ou STM
            │
            ▼
     ingester Go ───────► TimescaleDB
            │                    │
            │                    ▼
            └──────────────► API GraphQL Go
                                  │
                                  ▼
                         Vue 3 + dashboard SVG
```

- `services/ingester` décode, normalise et déduplique les snapshots protobuf.
- `db/migrations` crée les tables TimescaleDB, les contraintes et la vue des
  dernières positions.
- `services/api` expose uniquement les champs nécessaires au dashboard, avec
  limites de taille, complexité GraphQL, CORS explicite et endpoints de santé.
- `web` affiche les états chargement, erreur, vide, données vieillissantes et
  succès. La carte est un SVG schématique déterministe : aucune tuile externe
  n’est requise pour la démo publique.

## Vérifications locales

```powershell
go test ./...
go vet ./...
npm --prefix web ci
npm --prefix web run test -- --run
npm --prefix web run typecheck
npm --prefix web run build
docker compose -f deploy/compose/docker-compose.yml config
```

Pour les détails Compose, voir [deploy/compose/README.md](deploy/compose/README.md).
Les règles de contribution et de sécurité sont dans
[CONTRIBUTING.md](CONTRIBUTING.md) et [SECURITY.md](SECURITY.md).

## Mode STM réel

Le mode réel est volontairement opt-in. Copie `.env.example` vers `.env`, puis
configure localement `APP_ENV=stm` et `STM_API_KEY`. La clé reste côté
ingester et n’est jamais envoyée au navigateur ou incluse dans une image web.
Ne committe jamais `.env`, une clé, un payload réel ou des données personnelles.

## GTFS statique et attribution

Télécharge une archive après avoir vérifié son empreinte, puis importe-la dans
une base déjà migrée :

```powershell
pwsh -File scripts/download-gtfs.ps1 `
  -Url https://www.stm.info/sites/default/files/gtfs/gtfs_stm.zip `
  -Destination data/gtfs/gtfs_stm.zip `
  -ExpectedSha256 <sha256-fourni-au-moment-du-telechargement>
go run ./cmd/gtfsimport data/gtfs/gtfs_stm.zip 2026-09-12
```

Les versions importées sont immuables et une réimportation de la même version
est refusée. Voir [docs/data-sources.md](docs/data-sources.md) pour la source,
l’attribution et les conditions d’utilisation officielles STM.

## Statut du projet

Le jalon 1 est prêt pour une démonstration locale et un portfolio public. Il
ne prétend pas être un service de production : authentification, haute
disponibilité, rotation automatisée des secrets, rétention opérationnelle et
SLA restent hors périmètre.
