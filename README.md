# Ponctuel

[![CI](https://github.com/MinaSeliman1/Ponctuel/actions/workflows/ci.yml/badge.svg?branch=jalon-1)](https://github.com/MinaSeliman1/Ponctuel/actions/workflows/ci.yml)
[![Dépôt public](https://img.shields.io/badge/dépôt-public-2ea44f)](https://github.com/MinaSeliman1/Ponctuel)

Ponctuel mesure l’écart entre les prédictions d’arrivée des autobus de la STM
et les arrivées observées. Les jalons 1 à 25 fournissent une démonstration
complète, reproductible et gratuite : ingestion GTFS-Realtime, transport
Redpanda, matcher d’arrivée, mesure d’erreur TimescaleDB, API GraphQL,
predictor Python optionnel et dashboard Vue.

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

## Déploiement public gratuit

La démonstration complète peut être publiée gratuitement sur une VM OCI
Always Free, car elle conserve les volumes TimescaleDB et Redpanda nécessaires
au Compose. La procédure reproductible, le bootstrap de la VM et le workflow
de déploiement SSH sont dans [deploy/oci/README.md](deploy/oci/README.md).

Le déploiement public démarre en mode `fixture`, sans clé STM et sans service
payant. Le mode STM réel est opt-in et sa clé reste uniquement dans le `.env`
privé de la VM.

## Architecture

```text
GTFS-Realtime fixture ou STM
            │
            ▼
     ingester Go ───────► TimescaleDB
            │
            ├──────────────► Redpanda ───────► matcher Go
            │                                      │
            └──────────────────────────────────────┴──► TimescaleDB
                                                       │
                                                       ▼
                                              API GraphQL Go
                                                       │
                                                       ▼
                                              Vue 3 + dashboard SVG
```

- `services/ingester` décode, normalise et déduplique les snapshots protobuf.
- `services/bus` publie les événements versionnés dans `trip-updates` et
  `vehicle-positions`; `services/matcher` compare `last_update` et `geofence`
  et expose ses métriques Prometheus.
- `db/migrations` crée les tables TimescaleDB, les contraintes et la vue des
  dernières positions.
- `services/api` expose uniquement les champs nécessaires au dashboard, avec
  limites de taille, complexité GraphQL, CORS explicite et endpoints de santé.
- `web` affiche les états chargement, erreur, vide, données vieillissantes et
  succès. Le panneau de qualité regroupe les erreurs par horizon avec une
  moyenne pondérée par le nombre d’observations, puis fournit un tableau
  textuel en complément du graphique SVG accessible. La carte est un SVG
  schématique déterministe : aucune tuile externe n’est requise pour la démo
  publique. Le tableau des véhicules possède une recherche, des filtres réels
  par ligne et par retard, ainsi qu’une pagination locale accessible.
- Le dashboard propose un rafraîchissement manuel et automatique toutes les
  30 secondes; les données existantes restent visibles pendant la relance.
- Si une actualisation échoue, les dernières données valides restent visibles
  et un avertissement permet de relancer l’opération sans perdre la vue.
- Les préférences permettent de suspendre l’actualisation automatique sans
  désactiver le bouton de rafraîchissement manuel.
- La fraîcheur affichée évolue localement toutes les 30 secondes, même lorsque
  l’actualisation automatique est suspendue.
- Les panneaux Préférences et Filtres sont utilisables au clavier : le focus
  entre sur le premier contrôle, Échap ferme le panneau et le focus revient au
  bouton d’origine.
- Le tableau permet d’exporter en CSV les véhicules correspondant à la recherche
  et aux filtres actifs, directement dans le navigateur.
- Les champs texte exportés en CSV sont neutralisés contre les formules de
  tableur; les valeurs numériques restent exportées comme des nombres.
- La recherche et les filtres sont partageables via les paramètres de l’URL,
  sans rechargement ni stockage de données côté serveur.
- Le bouton « Copier le lien » permet de partager directement l’état courant
  des filtres depuis le tableau.
- Le tableau affiche aussi les filtres actifs sous forme de pastilles
  supprimables individuellement, avec une réinitialisation globale.
- Le compteur de filtres inclut la recherche et reste aligné sur les pastilles
  réellement affichées.
- Le tableau permet de trier localement les véhicules par ordre d’arrivée,
  ligne, identifiant ou retard décroissant.
- L’ordre de tri est également restauré et partagé via le paramètre URL `sort`.
- Le tableau se resynchronise aussi lors des actions retour/avance du navigateur.
- Les marqueurs de la carte sont sélectionnables à la souris ou au clavier et
  affichent une fiche accessible avec les détails du véhicule.
- Les contrôles de carte correspondent aux couches réellement disponibles et
  la légende explique les couleurs de retard des véhicules.
- La fiche d’un autobus se ferme avec Échap et restitue le focus au marqueur
  pour conserver un parcours clavier continu.

## Mesure et predictor

Le jalon 4 relie chaque arrivée à ses prédictions correspondantes et expose
les agrégats `errorSummary` par ligne et horizon. Le service Python
`services/predictor` entraîne un `GradientBoostingRegressor` avec séparation
temporelle et baseline zéro; il ne lit ni PostgreSQL, ni Redpanda, ni secret
STM. L’artefact Joblib est fourni séparément par un pipeline d’entraînement et
n’est pas versionné. La méthode, les limites et les conditions pour publier
un résultat réel sont documentées dans [docs/resultats.md](docs/resultats.md).

Le dashboard affiche cette mesure dans le panneau « Qualité des prédictions ».
Chaque horizon regroupe les résumés par ligne en conservant leur `sampleCount`;
une panne ou une absence de données de qualité ne bloque ni la carte ni la
liste des véhicules. Les fixtures permettent de vérifier le contrat visuel et
ne constituent pas une mesure STM.

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

Le parcours navigateur utilise Chromium et des fixtures GraphQL locales; il
ne contacte pas STM et ne mesure pas la disponibilité de l’API réelle :

```powershell
Push-Location web
npx playwright install chromium
npm run test:e2e
Pop-Location
```

Le smoke test Compose reste le contrôle d’intégration des services backend.

## Qualité du dépôt public

Chaque modification passe par une pull request et la CI vérifie les tests Go,
Python et frontend, le typage, le build, les tests navigateur, la configuration
Docker Compose, les images Docker, le chart Helm et la recherche de secrets.
Les dépendances GitHub Actions, Go, npm, Python et les images Docker sont
surveillées gratuitement par Dependabot. Les modèles d’issues et de pull
requests sont disponibles dans `.github/` pour garder les contributions
reproductibles.

Pour les détails Compose, voir [deploy/compose/README.md](deploy/compose/README.md).
Les règles de contribution et de sécurité sont dans
[CONTRIBUTING.md](CONTRIBUTING.md) et [SECURITY.md](SECURITY.md). Le chart
Kubernetes local et sa procédure de validation sont dans
[deploy/k8s/README.md](deploy/k8s/README.md).

## Mode STM réel

Le mode réel est volontairement opt-in. Copie `.env.example` vers `.env`, puis
configure localement `APP_ENV=stm` et `STM_API_KEY`. La clé reste côté
ingester et n’est jamais envoyée au navigateur ou incluse dans une image web.
Ne committe jamais `.env`, une clé, un payload réel ou des données personnelles.

## Déploiement public gratuit

Un profil Render + Supabase est disponible dans
[deploy/render/README.md](deploy/render/README.md). Il démarre en mode fixture
et conserve les données dans Supabase; aucun service payant ni clé STM n’est
nécessaire pour la première mise en ligne. Le profil Compose reste la
démonstration locale de la chaîne complète avec Redpanda, le matcher et
TimescaleDB.

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

Les jalons 1 à 25 sont prêts pour une démonstration locale, un déploiement k3s
documenté et un portfolio public. La démonstration complète reste locale et
gratuite. Le projet ne
prétend pas être un service de production : authentification, haute
disponibilité, rotation automatisée des secrets, rétention opérationnelle et
SLA restent hors périmètre.
