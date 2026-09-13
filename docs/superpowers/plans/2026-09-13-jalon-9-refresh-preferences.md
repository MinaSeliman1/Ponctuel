# Jalon 9 — implémentation des préférences de rafraîchissement

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-9-refresh-preferences-design.md`.

## Task 1 — Rendre Préférences fonctionnel

**Fichiers :**

- Modifier `web/src/App.vue`
- Modifier `web/src/styles.css`

- [x] Ajouter l’état d’ouverture et la préférence `autoRefreshEnabled`.
- [x] Extraire la gestion du timer pour pouvoir l’arrêter et le recréer.
- [x] Ajouter le dialogue accessible avec la case Actualisation automatique.
- [x] Documenter le réglage en mémoire dans le README.

## Task 2 — Tester le comportement

**Fichiers :**

- Modifier `web/tests/App.test.ts`

- [x] Vérifier l’ouverture du dialogue et la case cochée par défaut.
- [x] Vérifier que la désactivation affiche l’état suspendu.
- [x] Vérifier que le bouton Actualiser reste opérationnel après désactivation.

## Task 3 — Couvrir le parcours navigateur

**Fichiers :**

- Modifier `web/e2e/dashboard.spec.ts`

- [x] Ouvrir les préférences avec le clavier/role accessible.
- [x] Désactiver l’auto-refresh et conserver le bouton manuel disponible.
- [x] Vérifier une nouvelle série de trois requêtes après Actualiser.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, les artefacts et l’arbre Git.
- [x] Commiter sur `jalon-9`, pousser et ouvrir une PR vers `jalon-8`.
- [x] Attendre une CI verte sans fusion automatique.
