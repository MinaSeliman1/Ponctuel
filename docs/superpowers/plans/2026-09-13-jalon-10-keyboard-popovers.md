# Jalon 10 — implémentation des panneaux flottants accessibles

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-10-keyboard-popovers-design.md`.

## Task 1 — Gérer le focus de Préférences

**Fichiers :**

- Modifier `web/src/App.vue`

- [x] Ajouter les références du déclencheur et du panneau.
- [x] Focaliser le premier contrôle à l’ouverture.
- [x] Fermer avec Échap et restituer le focus au déclencheur.

## Task 2 — Gérer le focus des Filtres

**Fichiers :**

- Modifier `web/src/components/VehicleTable.vue`

- [x] Ajouter les références du déclencheur et du panneau.
- [x] Focaliser le sélecteur de ligne à l’ouverture.
- [x] Fermer avec Échap et restituer le focus au déclencheur.

## Task 3 — Tester le parcours clavier

**Fichiers :**

- Modifier `web/tests/App.test.ts`
- Modifier `web/tests/components/VehicleTable.test.ts`
- Modifier `web/e2e/dashboard.spec.ts`

- [x] Vérifier l’ouverture et le focus initial de Préférences.
- [x] Vérifier Échap et le focus restauré pour Préférences et Filtres.
- [x] Vérifier le parcours complet dans Chromium.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, les artefacts et l’arbre Git.
- [ ] Commiter sur `jalon-10`, pousser et ouvrir une PR vers `jalon-9`.
- [ ] Attendre une CI verte sans fusion automatique.
