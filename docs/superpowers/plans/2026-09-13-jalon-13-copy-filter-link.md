# Jalon 13 — implémentation du lien de filtres copiable

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-13-copy-filter-link-design.md`.

## Task 1 — Exposer une URL absolue et la copie Clipboard

**Fichiers :**

- Modifier `web/src/utils/vehicleFilters.ts`
- Modifier `web/tests/utils/vehicleFilters.test.ts`

- [x] Construire une URL absolue sans perdre les paramètres existants.
- [x] Copier le lien avec un échec explicite si Clipboard est indisponible.
- [x] Tester les états de succès et d’échec.

## Task 2 — Intégrer le bouton au tableau

**Fichiers :**

- Modifier `web/src/components/VehicleTable.vue`
- Modifier `README.md`

- [x] Ajouter le bouton accessible près de l’export CSV.
- [x] Afficher le succès ou l’échec dans un statut live.
- [x] Réinitialiser le statut quand les filtres changent.

## Task 3 — Couvrir le parcours utilisateur

**Fichiers :**

- Modifier `web/tests/components/VehicleTable.test.ts`
- Modifier `web/e2e/dashboard.spec.ts`

- [x] Vérifier la copie avec les filtres actifs.
- [x] Vérifier l’échec Clipboard et le reset du statut.
- [x] Vérifier le parcours complet dans Chromium.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, les artefacts et l’arbre Git.
- [x] Commiter sur `jalon-13`, pousser et ouvrir une PR vers `jalon-12`.
- [x] Attendre une CI verte sans fusion automatique.
