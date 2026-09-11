# Ponctuel - conception technique

Date : 2026-09-11  
Statut : approuvé pour planification et implémentation progressive  
Langue du produit et de la documentation : français

## 1. Objectif

Ponctuel mesure la qualité réelle des prédictions d'arrivée d'autobus de la STM.
Le produit conserve les versions successives des prédictions, reconstruit les
arrivées observées et publie l'erreur par ligne, horizon et période. Il pourra
ensuite comparer ces résultats à un modèle de machine learning, sans masquer un
résultat négatif.

Le projet vise une démonstration technique crédible : données réelles quand une
clé STM est configurée, fixtures reproductibles sans secret pour les tests et la
CI, architecture événementielle livrable par étapes, et documentation honnête
sur ce qui est démontré ou non.

## 2. Contraintes et décisions

- Aucun abonnement ou service payant ne sera requis pour le développement.
- Le dépôt sera public et ne contiendra aucune clé, donnée personnelle ou
  payload réel non nécessaire.
- Le mode `fixture` sera le chemin par défaut de la CI et de la démonstration
  locale minimale.
- Le mode STM sera activé par variables d'environnement, en particulier
  `STM_API_KEY`; la clé sera fournie localement ou comme secret GitHub, jamais
  dans le code.
- La collecte réelle sera limitée à une fréquence configurable, avec 30 secondes
  comme valeur de départ et un backoff sur les erreurs.
- Une démo web statique gratuite pourra utiliser des snapshots fictifs ou
  anonymisés. Une ingestion live 24/7 ne sera annoncée que si un hébergement
  gratuit réellement vérifié existe; sinon le dépôt fournira le parcours local
  complet.
- L'utilisation des données STM devra attribuer la source et respecter les
  conditions d'utilisation. Le logo STM ne sera pas utilisé sans autorisation.

## 3. Architecture cible

Le dépôt est un monorepo organisé par responsabilités :

```text
ponctuel/
├── services/
│   ├── ingester/       # Go: STM GTFS-Realtime -> événements normalisés
│   ├── matcher/        # Go: horaire + événements -> arrivées et erreurs
│   ├── api/            # Go: GraphQL, santé et métriques
│   └── predictor/      # Python/FastAPI: entraînement et prédiction
├── web/                # Vue 3 + Vite + TypeScript
├── internal/           # contrats et bibliothèques Go partagés
├── db/migrations/      # migrations SQL versionnées
├── deploy/
│   ├── compose/        # développement et démonstration locale
│   └── k8s/            # manifestes et chart Helm du jalon 3
├── docs/
│   ├── superpowers/specs/
│   ├── architecture.md
│   ├── methode.md
│   ├── resultats.md
│   └── journal.md
├── testdata/           # petites fixtures protobuf et GTFS
├── .github/
│   ├── workflows/
│   └── ISSUE_TEMPLATE/
└── README.md
```

Au début, les services Go partageront un module racine pour réduire la
complexité de développement tout en produisant des binaires séparés. Les
frontières processus seront réelles dans Docker Compose puis Kubernetes. Une
extraction en modules indépendants ne sera faite que si elle apporte une
isolation mesurable.

### Flux de données

```text
STM GTFS-Realtime
      │ protobuf, 30 s
      ▼
   ingester ── raw feed + événements normalisés ──► stockage
      │
      └─ jalon 2 : Redpanda topics trip-updates / vehicle-positions
                                                    │
                                                    ▼
                                                 matcher
                                                    │
                         prediction / arrival_observed / prediction_error
                                                    │
                                                    ▼
                                                API Go
                                                    │
                                                    ▼
                                                Vue web
```

Redpanda sera introduit au jalon 2, car l'ingestion et le calcul d'arrivée ont
des profils de panne et de charge différents. Le contrat d'événements sera
défini dès le jalon 1 pour que ce changement ne modifie pas les règles métier.

## 4. Modèle de données

Les tables statiques GTFS nécessaires sont `routes`, `trips`, `stops` et
`stop_times`, avec les colonnes utiles de `calendar` et `calendar_dates`.
Elles servent de référence planifiée et sont rechargées avec une version de
feed identifiée.

Les tables propres au projet sont :

- `feed_snapshot` : type de feed, heure de collecte, timestamp source, hash du
  payload, nombre d'entités, statut de décodage et chemin local optionnel du
  payload brut. L'unicité sur `(feed_type, payload_hash)` garantit
  l'idempotence.
- `vehicle_position` : position observée, véhicule, voyage, arrêt courant,
  timestamp véhicule et identifiant de snapshot. C'est une hypertable.
- `prediction` : une ligne par prédiction publiée, sans écrasement des versions
  antérieures; elle contient `recorded_at`, `trip_id`, `service_date`, `route_id`,
  `stop_id`, `stop_sequence`, `predicted_at`, `delay_s`, `vehicle_id` et
  `horizon_s` dérivé. C'est une hypertable.
- `arrival_observed` : arrivée déduite avec `method` (`last_update` ou
  `geofence`), `confidence`, `observed_at`, `trip_id`, `service_date` et
  `stop_id`. La clé logique évite deux arrivées concurrentes pour la même
  instance de voyage et le même arrêt.

La vue `prediction_error` joint les prédictions aux arrivées observées et
calcule `observed_at - predicted_at`. Les analyses utilisent des horizons
non-négatifs limités à une heure et conservent la méthode de déduction. La
catégorie « à l'heure » est asymétrique : de 60 secondes en avance à 300
secondes en retard.

Les données brutes sont conservées dans un volume local ou un stockage externe
optionnel, jamais dans Git. Une politique de rétention configurée évite qu'une
démo locale remplisse le disque; les agrégats analytiques peuvent être gardés
plus longtemps que les payloads bruts.

## 5. Ingester et intégration STM

L'ingester expose une configuration explicite :

- `STM_TRIP_UPDATES_URL`;
- `STM_VEHICLE_POSITIONS_URL`;
- `STM_API_KEY`;
- `STM_API_KEY_HEADER` avec `apikey` comme valeur initiale vérifiable dans le
  Swagger du portail;
- `POLL_INTERVAL` avec 30 secondes par défaut;
- `DATABASE_URL` et `INGEST_MODE=fixture|stm`.

L'implémentation ne supposera pas que l'API renvoie JSON : les réponses seront
validées comme protobuf GTFS-Realtime et décodées avec les bindings Go
MobilityData. Le parseur normalisera les entités vers des structures internes
indépendantes du client HTTP.

Comportement réseau :

- `401`/`403` : erreur de configuration, sans retry agressif et avec message
  actionnable sans afficher la clé;
- `429` : respect de `Retry-After` si présent, puis backoff exponentiel borné;
- `5xx`, timeout ou coupure : retry borné, métrique d'erreur et reprise au cycle
  suivant;
- protobuf invalide : snapshot marqué en échec et payload placé en quarantaine
  locale, sans interrompre définitivement le service;
- arrêt : contexte annulé, fermeture des connexions et flush des métriques.

Le premier smoke test réel sera fait avec la clé configurée localement et une
requête manuelle limitée. Il vérifiera l'URL, le nom exact du header, le statut,
le content-type et la présence d'entités avant d'activer la boucle continue.

## 6. Matcher et règles d'arrivée

Le matcher consommera des événements normalisés et joindra les identifiants
réels aux tables GTFS statiques. Il calculera les délais en secondes en
respectant la convention GTFS : négatif signifie en avance.

Deux stratégies seront implémentées et comparées :

1. `last_update` : dernière prédiction observée avant que l'arrêt disparaisse
   du flux ou soit dépassé;
2. `geofence` : franchissement d'un rayon autour de l'arrêt à partir des
   positions de véhicule.

Chaque résultat sera accompagné d'une confiance et de raisons de rejet. Les
cas ambigus ne seront pas utilisés silencieusement dans les statistiques; ils
seront comptés et documentés dans `docs/methode.md`.

## 7. API, interface et observabilité

L'API Go fournira un schéma GraphQL versionné en lecture pour : lignes, arrêts,
positions récentes, volumes ingérés, distributions d'erreur et précision par
horizon. Elle fournira aussi :

- `/healthz` : processus vivant;
- `/readyz` : configuration valide et connexion DB fonctionnelle;
- `/metrics` : métriques Prometheus.

Les métriques minimales sont le nombre de feeds reçus, entités décodées,
insertions, doublons, erreurs par type, latence de collecte, fraîcheur du feed,
retard de consommation Redpanda et disponibilité de la base.

L'interface Vue sera d'abord une page de démonstration claire et responsive :

- bandeau d'attribution STM et statut du mode (`fixture` ou `STM`);
- compteur d'événements et dernière collecte;
- carte des véhicules quand les positions sont disponibles;
- tableau filtrable par ligne;
- graphique d'erreur par horizon;
- états chargement, erreur, vide et données vieillissantes.

La carte utilisera des tuiles et une librairie dont la licence et le quota sont
documentés; aucun logo STM ne sera intégré sans autorisation.

## 8. Sécurité et exploitation

- `.env.example` documente les variables; `.env` est ignoré.
- Les logs ne contiennent ni clé, ni headers d'authentification, ni payload
  complet.
- Les endpoints de collecte restent internes; l'API publique est en lecture.
- Les entrées de filtres sont validées et bornées; les requêtes GraphQL ont une
  profondeur et un coût maximum documentés avant exposition publique.
- Les dépendances sont verrouillées et auditées par la CI.
- Les manifestes Kubernetes utilisent Secret pour la clé, ConfigMap pour la
  configuration non sensible, probes séparées et permissions minimales.
- Les sauvegardes/restaurations et la rétention sont documentées pour le mode
  local; aucune fausse garantie de sauvegarde cloud ne sera annoncée.

## 9. Jalons et critères d'acceptation

### Jalon 1 - ingérer et afficher

- Go ingester en mode fixture et STM;
- import GTFS statique reproductible;
- migrations TimescaleDB et stockage idempotent;
- tests du parseur, normalisateur et repository;
- API minimale de compteur/positions;
- Vue minimale et carte;
- Docker Compose et README de démarrage;
- CI sans secret obligatoire.

### Jalon 2 - séparer et fiabiliser

- Redpanda local;
- topics `trip-updates` et `vehicle-positions`;
- matcher séparé;
- deux méthodes d'arrivée comparées;
- métriques Prometheus dans chaque service;
- test d'intégration Compose.

### Jalon 3 - déployer sur Kubernetes

- images versionnées;
- chart Helm avec `values.yaml` et schéma de valeurs;
- Deployments/Services, Secret, ConfigMap, StatefulSet DB;
- startup, liveness et readiness probes;
- pipeline de build et validation des manifestes.

### Jalon 4 - mesurer puis essayer de battre

- agrégations par horizon, ligne, heure et condition;
- validation temporelle sans mélange futur/passé;
- modèle gradient boosting reproductible;
- rapport `docs/resultats.md` et graphique public;
- verdict publié même si le modèle ne bat pas STM.

## 10. Stratégie de tests et CI

- Go : `go test ./...`, tests de propriétés sur les délais et tests HTTP avec
  `httptest`;
- SQL : migrations sur une base éphémère et vérification des contraintes;
- intégration : Compose démarre la DB, le service et les fixtures;
- Python : tests de validation des données, entraînement temporel et contrat
  `/predict`;
- web : typecheck, unit tests Vitest et parcours navigateur Playwright à partir
  du jalon 2;
- qualité : formatage, lint, audit de dépendances, build Docker et validation
  Helm;
- CI : jobs séparés, permissions GitHub minimales, secrets STM optionnels et
  jamais requis pour une pull request.

## 11. Risques et limites assumés

- La documentation publique STM ne décrit pas toutes les limites propres à
  chaque API; les quotas officiels connus seront configurés conservativement et
  toute réponse 429 sera observée.
- Le portail STM est une application JavaScript authentifiée; le Swagger et le
  nom exact du header seront vérifiés dans le premier smoke test, pas inventés.
- Une disponibilité 24/7 gratuite pour une stack Redpanda + TimescaleDB + k3s
  n'est pas garantie. Le dépôt sera complet et démontrable localement avant de
  chercher un hébergement gratuit.
- La qualité des arrivées observées dépend des données disponibles et sera
  quantifiée; les résultats ne seront pas présentés comme vérité terrain
  parfaite.
- Les limites légales d'utilisation des données STM doivent rester compatibles
  avec les conditions officielles en vigueur au moment de la publication.

## 12. Sources officielles consultées

- STM, portail et quotas :
  https://www.stm.info/fr/a-propos/developpeurs/faq-nouveau-portail-developpeurs
- STM, données développeurs :
  https://www.stm.info/fr/a-propos/developpeurs
- STM, conditions d'utilisation :
  https://www.stm.info/fr/a-propos/developpeurs/condition-dutilisation
- GTFS-Realtime, bonnes pratiques :
  https://gtfs.org/documentation/realtime/realtime-best-practices/
- GTFS-Realtime, Trip Updates :
  https://gtfs.org/documentation/realtime/feed-entities/trip-updates/
- MobilityData, bindings :
  https://github.com/MobilityData/gtfs-realtime-bindings
- Redpanda, démarrage self-managed :
  https://docs.redpanda.com/streaming/current/get-started/quick-start/
- TimescaleDB, hypertables : https://docs.timescale.com/use-timescale/latest/hypertables/
- TimescaleDB, rétention :
  https://docs.timescale.com/use-timescale/latest/data-retention/create-a-retention-policy/
- GraphQL : https://graphql.org/
- Vue + TypeScript : https://vuejs.org/guide/typescript/overview.html
- FastAPI : https://fastapi.tiangolo.com/tutorial/first-steps/
- Prometheus : https://prometheus.io/docs/instrumenting/exposition_formats/
- Kubernetes, probes : https://kubernetes.io/docs/concepts/workloads/pods/probes/
- Helm, charts : https://helm.sh/docs/topics/charts/
- GitHub Actions, coûts publics :
  https://docs.github.com/en/actions/concepts/billing-and-usage
- GitHub Actions, secrets :
  https://docs.github.com/en/actions/concepts/security/secrets
