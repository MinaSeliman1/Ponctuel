# Journal des changements

Toutes les modifications importantes du projet sont consignées ici. Les
jalons sont développés sur une branche dédiée, vérifiés par la CI, puis
fusionnés dans `jalon-1`.

## [1.0.0] — 2026-09-13

Première version publique présentable dans un portfolio.

- ingestion GTFS-Realtime en mode fixture gratuit ou STM réel opt-in;
- pipeline Go avec Redpanda, matcher d’arrivées et TimescaleDB;
- API GraphQL avec limites, santé, readiness, métriques et CORS explicite;
- predictor Python avec séparation temporelle et comparaison à une baseline;
- dashboard Vue accessible avec carte schématique, filtres, tri, pagination,
  partage d’URL, export CSV et gestion des rafraîchissements en erreur;
- tests unitaires, intégration base de données, E2E Chromium, tests Python,
  validation Docker/Compose et Helm;
- protection des secrets, neutralisation des formules CSV et documentation
  publique des limites de la démonstration;
- déploiement local k3s documenté, sans service payant obligatoire.

## Prochaines évolutions possibles

Les sujets volontairement hors périmètre de cette version sont
l’authentification, la haute disponibilité, la rotation automatisée des
secrets, la rétention opérationnelle et un SLA de production. Ils devront être
traités avec une infrastructure et des exigences métier explicites.
