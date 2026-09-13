# Jalon 22 — plan d’implémentation de la résilience du rafraîchissement

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-22-refresh-failure-design.md`.

## Task 1 — Séparer les états de chargement et de relance

**Fichiers :**

- Modifier `web/src/App.vue`
- Modifier `web/src/components/StatusPanel.vue`
- Modifier `web/src/styles.css`

- [x] Ajouter l’état `refreshError` et un contrôle commun des annulations.
- [x] Préserver les payloads existants pendant une relance échouée.
- [x] Exposer un avertissement accessible et une action « Réessayer ».
- [x] Conserver le comportement du premier chargement en erreur.

## Task 2 — Tester le contrat frontend

**Fichiers :**

- Modifier `web/tests/App.test.ts`

- [x] Simuler une première collecte réussie puis trois réponses 503.
- [x] Vérifier que la table, le véhicule et l’avertissement restent présents.
- [x] Vérifier que l’action de nouvelle tentative reste disponible.

## Task 3 — Couvrir le parcours navigateur

**Fichiers :**

- Modifier `web/e2e/dashboard.spec.ts`

- [x] Faire réussir la première série de requêtes et échouer la seconde.
- [x] Vérifier la conservation du véhicule et l’annonce accessible.
- [x] Vérifier que le bouton « Réessayer » reste activé.

## Task 4 — Documenter et valider

**Fichiers :**

- Modifier `README.md`
- Ajouter cette spécification et ce plan

- [x] Documenter le comportement non destructif dans le README.
- [x] Exécuter tests frontend, typecheck, build et E2E.
- [x] Exécuter les contrôles backend, Compose, Helm et smoke.
- [x] Vérifier le diff, commiter sur `jalon-22`, pousser et ouvrir une PR vers
  `jalon-21`.
- [x] Attendre la CI de la PR sans fusion automatique.
