# Jalon 5 Dashboard qualité Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use Markdown checkboxes for tracking.

**Goal:** Afficher dans le dashboard la qualité des prédictions par horizon avec un graphique SVG accessible, reproductible et honnête.

**Architecture:** Le client Vue interroge le champ GraphQL `errorSummary` déjà livré au jalon 4. Le composant `ErrorQualityChart` agrège côté interface les résumés par horizon avec une moyenne pondérée par `sampleCount`, puis rend une courbe SVG sans librairie cartographique, tuile externe ou service payant. L’échec de cette métrique reste indépendant de la carte et du tableau des véhicules afin que le dashboard demeure utile quand aucune erreur n’est encore calculable.

**Tech Stack:** Vue 3.5, TypeScript, Vitest, SVG natif, GraphQL HTTP existant.

**Spec:** `docs/superpowers/specs/2026-09-11-ponctuel-design.md`, section 7 et critères d’interface du jalon 4.

## Global Constraints

- Ne jamais présenter les fixtures comme une mesure STM réelle.
- Conserver les états chargement, erreur, vide, données vieillissantes et succès.
- Limiter la requête à 100 résumés et ne pas exposer de nouvelle clé ou dépendance payante.
- Respecter l’accessibilité : titre SVG, résumé textuel, tableau lisible et focus visible.
- Ne pas faire échouer le dashboard positions si `errorSummary` est indisponible.
- Les moyennes affichées doivent être pondérées par `sampleCount`; ne jamais moyenner naïvement des moyennes de groupes.

---

### Task 1: Contrat frontend et requête GraphQL

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api/client.ts`
- Create: `web/tests/api/client.test.ts`

**Interfaces:**
- Produit `ErrorSummary` avec `routeId: string | null`, `horizonSeconds: number`, `sampleCount: number`, `meanErrorSeconds: number`, `onTimeRate: number`.
- Produit `fetchErrorSummary(limit = 100, signal?)` qui envoie `errorSummary(limit: $limit)` et retourne `Promise<ErrorSummary[]>`.

- [x] **Step 1: Write the failing client tests**

Tester avec un `fetch` mocké que la requête demande les cinq champs, envoie la variable `limit: 100`, mappe deux lignes et propage une réponse HTTP non-OK comme erreur.

- [x] **Step 2: Run the focused tests and verify failure**

Run: `npm --prefix web run test -- --run tests/api/client.test.ts`

Expected: FAIL because `fetchErrorSummary` and `ErrorSummary` do not exist.

- [x] **Step 3: Implement the typed query**

Réutiliser le helper GraphQL `query`, ajouter le type et borner localement `limit` à 100 avant l’envoi.

- [x] **Step 4: Run the focused tests**

Run: `npm --prefix web run test -- --run tests/api/client.test.ts`

Expected: PASS.

- [x] **Step 5: Commit**

```powershell
git add web/src/types.ts web/src/api/client.ts web/tests/api/client.test.ts
git commit -m "feat: fetch prediction quality summaries"
```

### Task 2: Composant SVG de qualité

**Files:**
- Create: `web/src/components/ErrorQualityChart.vue`
- Create: `web/tests/components/ErrorQualityChart.test.ts`
- Modify: `web/src/styles.css`

**Interfaces:**
- Props: `summaries: ErrorSummary[]`, `isLoading: boolean`, `hasError: boolean`, `stale: boolean`.
- Agrégation interne par horizon: `sampleCount = sum(sampleCount)`, `meanErrorSeconds = sum(meanErrorSeconds * sampleCount) / sampleCount`, `onTimeRate = sum(onTimeRate * sampleCount) / sampleCount`.
- Affiche un SVG `viewBox="0 0 760 300"`, une ligne zéro, axes libellés, points reliés par horizon et tableau textuel de synthèse.

- [x] **Step 1: Write failing component tests**

Tester les états loading/error/empty/stale, l’agrégation pondérée de deux routes au même horizon, la présence du titre SVG accessible, les valeurs `+42 s`, `−15 s` et `75 %` dans le résumé textuel, ainsi que l’absence de contenu SVG quand aucune observation n’est disponible.

- [x] **Step 2: Run focused tests and verify failure**

Run: `npm --prefix web run test -- --run tests/components/ErrorQualityChart.test.ts`

Expected: FAIL because the component is absent.

- [x] **Step 3: Implement the component**

Créer les échelles déterministes, utiliser une palette existante (`--green`, `--amber`, `--red`), protéger les divisions par zéro, limiter les labels à quelques horizons, et conserver un tableau HTML pour les lecteurs d’écran et petits écrans.

- [x] **Step 4: Add responsive styles and pass tests**

Ajouter les classes `.quality-panel`, `.quality-chart`, `.quality-table` et leurs règles responsive sans modifier les styles carte/tableau existants.

Run: `npm --prefix web run test -- --run tests/components/ErrorQualityChart.test.ts`

Expected: PASS.

- [x] **Step 5: Commit**

```powershell
git add web/src/components/ErrorQualityChart.vue web/tests/components/ErrorQualityChart.test.ts web/src/styles.css
git commit -m "feat: render accessible prediction quality chart"
```

### Task 3: Intégration résiliente au dashboard

**Files:**
- Modify: `web/src/App.vue`
- Create: `web/tests/App.test.ts`

**Interfaces:**
- `loadData` utilise trois résultats indépendants: dashboard, véhicules, résumés qualité.
- Une erreur qualité active `qualityError` sans remplacer les données valides de la carte et du tableau.
- `ErrorQualityChart` reçoit `dashboard?.stale` et le bouton de rafraîchissement existant relance la requête.

- [x] **Step 1: Write failing App integration tests**

Tester que la page rend simultanément la carte/tableau et le graphique quand les trois requêtes réussissent, qu’une erreur `errorSummary` affiche le message qualité tout en conservant le tableau, et qu’un résultat vide affiche l’état « pas encore assez d’observations ».

- [x] **Step 2: Run focused tests and verify failure**

Run: `npm --prefix web run test -- --run tests/App.test.ts`

Expected: FAIL because `App.vue` ne charge pas encore `errorSummary`.

- [x] **Step 3: Integrate with `Promise.allSettled`**

Conserver le comportement actuel pour dashboard/véhicules, traiter séparément le rejet qualité, annuler le contrôleur au démontage, et placer le composant après la carte mais avant le tableau.

- [x] **Step 4: Run the complete frontend verification**

Run: `npm --prefix web run test -- --run`, `npm --prefix web run typecheck`, `npm --prefix web run build`

Expected: all Vitest tests, typecheck and production build pass.

- [x] **Step 5: Commit**

```powershell
git add web/src/App.vue web/tests/App.test.ts
git commit -m "feat: integrate prediction quality into dashboard"
```

### Task 4: Documentation, CI et publication

**Files:**
- Modify: `README.md`
- Modify: `docs/resultats.md`
- Modify: `docs/superpowers/plans/2026-09-12-jalon-5-dashboard-qualite.md`

- [x] **Step 1: Document the UI contract**

Décrire le nouveau panneau, l’agrégation pondérée, les états sans données et le fait qu’un graphique fixture n’est pas une performance STM.

- [x] **Step 2: Mark the plan complete and run repository checks**

Cocher toutes les étapes puis exécuter `scripts/check.ps1`, `scripts/smoke.ps1`, `go vet ./...`, `git diff --check` et vérifier qu’aucune clé STM n’apparaît dans les assets web.

- [x] **Step 3: Commit, push and open the PR**

```powershell
git add README.md docs/resultats.md docs/superpowers/plans/2026-09-12-jalon-5-dashboard-qualite.md
git commit -m "docs: document dashboard prediction quality"
git push -u origin jalon-5
gh pr create --base jalon-4 --head jalon-5 --title "feat: add prediction quality dashboard"
```

La PR cible `jalon-4` et ne doit pas être fusionnée automatiquement.
