# Sources de données

## GTFS statique STM

- Source officielle : [page Développeurs STM](https://www.stm.info/fr/a-propos/developpeurs)
- Fichier de téléchargement observé le 12 septembre 2026 : [gtfs_stm.zip](https://www.stm.info/sites/default/files/gtfs/gtfs_stm.zip)
- Description des fichiers : [Description des données disponibles](https://www.stm.info/fr/a-propos/developpeurs/description-des-donnees-disponibles)
- Conditions d'utilisation : [conditions officielles STM](https://www.stm.info/fr/node/4185)

L'URL et le contenu du fichier peuvent changer avec les périodes de service. Le
script `scripts/download-gtfs.ps1` exige donc un SHA-256 fourni au moment du
téléchargement et refuse d'écrire hors de `data/`.

L'import conserve une version immuable identifiée par `feed_version`. Une
seconde importation de la même version est rejetée afin de ne pas remplacer
silencieusement l'horaire utilisé pour une analyse.

Ponctuel attribue les données à la Société de transport de Montréal et
n'utilise pas son logo. Les données sont utilisées selon les conditions de la
STM; ce dépôt ne redistribue pas une archive GTFS réelle.
