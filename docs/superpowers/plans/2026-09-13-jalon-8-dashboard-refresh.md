# Jalon 8 — implémentation du rafraîchissement du dashboard

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-8-dashboard-refresh-design.md`.

## Task 1 — Séparer chargement initial et rafraîchissement

**Fichiers :**

- Modifier `web/src/App.vue`
- Modifier `web/src/components/StatusPanel.vue`
- Modifier `web/src/styles.css`

- [x] Ajouter `isRefreshing`, un intervalle de 30 secondes et son nettoyage.
- [x] Conserver les données visibles pendant une requête de rafraîchissement.
- [x] Rendre le bouton Actualiser accessible, désactivé pendant l’appel et
  compatible avec l’état d’erreur existant.
- [x] Documenter le comportement dans le README.

## Task 2 — Tester l’interaction locale

**Fichiers :**

- Modifier `web/tests/App.test.ts`
- Modifier `web/tests/components/StatusPanel.test.ts`

- [x] Vérifier que le bouton émet l’événement `refresh`.
- [x] Vérifier qu’un clic déclenche une nouvelle série de requêtes sans perdre
  le véhicule affiché.
- [x] Vérifier que l’état `Actualisation…` est annoncé et désactive le bouton.

## Task 3 — Tester le parcours navigateur

**Fichiers :**

- Modifier `web/e2e/dashboard.spec.ts`

- [x] Compter les requêtes fixture dans le routeur Playwright.
- [x] Cliquer sur Actualiser et vérifier que le dashboard recharge ses trois
  contrats GraphQL.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, les artefacts et l’arbre Git.
- [ ] Commiter sur `jalon-8`, pousser et ouvrir une PR vers `jalon-7`.
- [ ] Attendre une CI verte sans fusion automatique.
