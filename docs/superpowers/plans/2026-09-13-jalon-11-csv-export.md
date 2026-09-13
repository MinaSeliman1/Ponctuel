# Jalon 11 — implémentation de l’export CSV

> Plan basé sur `docs/superpowers/specs/2026-09-13-jalon-11-csv-export-design.md`.

## Task 1 — Ajouter la sérialisation CSV

**Fichiers :**

- Créer `web/src/utils/csv.ts`

- [x] Ajouter l’échappement CSV et les colonnes véhicules stables.
- [x] Ajouter la fonction de téléchargement local avec `Blob`.
- [x] Tester les virgules, guillemets, accents et valeurs nulles.

## Task 2 — Brancher l’export au tableau

**Fichiers :**

- Modifier `web/src/components/VehicleTable.vue`
- Modifier `web/src/styles.css`

- [x] Ajouter le bouton accessible et ses états désactivés.
- [x] Exporter toutes les lignes filtrées, indépendamment de la pagination.
- [x] Documenter la capacité dans le README.

## Task 3 — Couvrir le parcours utilisateur

**Fichiers :**

- Modifier `web/tests/components/VehicleTable.test.ts`
- Créer `web/tests/utils/csv.test.ts`
- Modifier `web/e2e/dashboard.spec.ts`

- [x] Vérifier la sérialisation et le bouton export.
- [x] Vérifier que le fichier ne contient que les véhicules filtrés.
- [x] Vérifier le téléchargement réel dans Chromium.

## Task 4 — Valider et publier

- [x] Exécuter tests, typecheck, build, E2E, `scripts/check.ps1` et smoke.
- [x] Vérifier `git diff --check`, les artefacts et l’arbre Git.
- [x] Commiter sur `jalon-11`, pousser et ouvrir une PR vers `jalon-10`.
- [x] Attendre une CI verte sans fusion automatique.
