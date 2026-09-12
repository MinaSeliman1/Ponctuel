# Ponctuel

Ponctuel mesure l'écart entre les prédictions d'arrivée des autobus de la STM
et les arrivées observées. Le projet est construit par jalons : la première
version ingère des feeds GTFS-Realtime, conserve les événements et affiche les
positions sur une carte.

## État actuel

Le dépôt est en construction. Le jalon 1 est développé en mode fixture-first :
les tests et la CI ne nécessitent aucune clé STM. Le mode STM réel sera activé
localement avec `STM_API_KEY` et ne sera pas présenté comme une démo publique
avant d'avoir été vérifié de bout en bout.

## Démarrage prévu

```powershell
Copy-Item .env.example .env
docker compose -f deploy/compose/docker-compose.yml up --build
```

Les détails d'exécution et les limites seront ajoutés avec le jalon 1. Ne
committe jamais `.env`, une clé STM ou un payload réel contenant des données
non nécessaires.

## Import GTFS statique

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
l'attribution et les conditions d'utilisation.

## Données STM

Les données STM sont utilisées selon les conditions officielles et devront être
attribuées dans l'interface et la documentation. Ce projet n'utilise pas le
logo de la STM. Sources :

- https://www.stm.info/fr/a-propos/developpeurs
- https://www.stm.info/fr/a-propos/developpeurs/condition-dutilisation
