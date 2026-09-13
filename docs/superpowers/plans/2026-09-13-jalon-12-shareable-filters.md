# Jalon 12 — implémentation des filtres partageables

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-12-shareable-filters-design.md`.

## Task 1 — Encapsuler l’état d’URL

**Fichiers :**

- Créer `web/src/utils/vehicleFilters.ts`
- Créer `web/tests/utils/vehicleFilters.test.ts`

- [x] Lire et valider `q`, `route` et `delay`.
- [x] Écrire les paramètres avec `history.replaceState`.
- [x] Tester les valeurs absentes, invalides et l’effacement.

## Task 2 — Synchroniser le tableau

**Fichiers :**

- Modifier `web/src/components/VehicleTable.vue`
- Modifier `README.md`

- [x] Initialiser la recherche et les filtres depuis l’URL.
- [x] Mettre à jour l’URL à chaque changement sans navigation.
- [x] Préserver la pagination, le CSV et les états accessibles.

## Task 3 — Couvrir le partage navigateur

**Fichiers :**

- Modifier `web/tests/components/VehicleTable.test.ts`
- Modifier `web/e2e/dashboard.spec.ts`

- [x] Vérifier la restauration de l’état au montage.
- [x] Vérifier la mise à jour de l’URL après une interaction.
- [x] Vérifier le parcours complet dans Chromium.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, les artefacts et l’arbre Git.
- [x] Commiter sur `jalon-12`, pousser et ouvrir une PR vers `jalon-11`.
- [x] Attendre une CI verte sans fusion automatique.
