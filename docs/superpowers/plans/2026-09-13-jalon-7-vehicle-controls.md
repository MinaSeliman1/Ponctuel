# Jalon 7 — implémentation des contrôles du tableau

> Plan d’implémentation du tableau des véhicules, basé sur la spec
> `docs/superpowers/specs/2026-09-13-jalon-7-vehicle-controls-design.md`.

## Task 1 — Rendre les filtres et la pagination fonctionnels

**Fichiers :**

- Modifier `web/src/components/VehicleTable.vue`
- Modifier `web/src/styles.css`

- [x] Ajouter les états `filtersOpen`, `selectedRoute`, `selectedDelay` et la
  page courante, avec des computed pour les valeurs de lignes, le résultat
  filtré, le nombre de pages et la tranche affichée.
- [x] Remplacer le bouton décoratif par un panneau accessible contenant les
  deux filtres, un bouton de réinitialisation et un bouton de fermeture.
- [x] Remplacer les pages fictives par une pagination de 10 lignes avec
  précédent/suivant, libellé de page et compteurs cohérents.
- [x] Ajouter les styles responsive du panneau et des contrôles.

## Task 2 — Tester le comportement dans Vitest

**Fichiers :**

- Modifier `web/tests/components/VehicleTable.test.ts`

- [x] Tester l’ouverture du panneau, le filtrage par ligne et la remise à zéro.
- [x] Tester le filtre de retard important et l’état vide associé.
- [x] Tester la pagination avec 11 véhicules et la navigation entre les pages.
- [x] Vérifier que les tests existants de recherche et de rendu restent verts.

## Task 3 — Couvrir le parcours public dans Playwright

**Fichiers :**

- Modifier `web/e2e/dashboard.spec.ts`

- [x] Ajouter une fixture de plusieurs véhicules avec deux lignes et des
  retards différents.
- [x] Vérifier dans Chromium qu’un filtre de ligne réduit le tableau et qu’un
  filtre de retard masque les véhicules à l’heure.
- [x] Vérifier que la pagination affiche la page 2 et le véhicule suivant.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, le scan de secrets et un arbre propre.
- [x] Commiter sur `jalon-7`, pousser la branche et ouvrir une PR vers `jalon-6`.
- [x] Attendre une CI verte sans fusion automatique.
