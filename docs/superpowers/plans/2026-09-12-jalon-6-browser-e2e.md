# Jalon 6 Browser E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use Markdown checkboxes for tracking.

**Goal:** Vérifier le dashboard Ponctuel dans Chromium avec des fixtures GraphQL locales, sans clé STM ni dépendance à un service externe.

**Architecture:** Playwright démarre le serveur Vite de web, puis les tests interceptent POST /query avec des réponses déterministes pour le dashboard, les véhicules et errorSummary. Le scénario nominal vérifie les sections accessibles et le scénario dégradé vérifie qu’une erreur de qualité n’empêche pas l’affichage des véhicules. GitHub Actions exécute ce parcours dans un job séparé après installation de Chromium.

**Tech Stack:** Playwright Test 1.63.0, Chromium, Vite, Vue 3.5, TypeScript, GitHub Actions, Node 24.

**Spec:** docs/superpowers/specs/2026-09-12-jalon-6-browser-e2e-design.md, plus la section 10 de docs/superpowers/specs/2026-09-11-ponctuel-design.md.

## Global Constraints

- Les scénarios E2E n’appellent aucun domaine externe et n’utilisent jamais STM_API_KEY.
- Les fixtures servent uniquement au contrat de test et ne sont pas présentées comme des mesures STM.
- Le navigateur reste une dépendance de test; aucune image de production ne l’installe.
- Les assertions utilisent les rôles et noms accessibles avant les classes CSS.
- Le job CI E2E reste séparé des tests Vitest, du typecheck, du build et du smoke Compose.
- Le projet conserve Node 24 et les dépendances verrouillées dans web/package-lock.json.

---

### Task 1: Installer Playwright et créer le parcours nominal

**Files:**
- Modify: web/package.json
- Modify: web/package-lock.json
- Create: web/playwright.config.ts
- Create: web/e2e/dashboard.spec.ts

**Interfaces:**
- Produit le script npm run test:e2e depuis web.
- Produit une configuration Chromium avec baseURL http://127.0.0.1:4173 et un webServer Vite.
- Produit une fixture GraphQL locale pour dashboard, vehicles et errorSummary.

- [x] **Step 1: Écrire le test nominal avant l’implémentation**

Créer web/e2e/dashboard.spec.ts avec une fonction routeDashboard(page) qui intercepte **/query, inspecte request.postDataJSON().query et répond avec un dashboard STM de 24 événements, un autobus 1234 de la ligne 51, et deux résumés au même horizon: 60 secondes avec 4 observations et -30 secondes avec 2 observations. Le test ouvre / et vérifie les rôles heading Qualité des prédictions et Véhicules en service, l’image accessible Erreur moyenne par horizon, la ligne 1234, le texte +30 s et le statut Temps réel.

- [x] **Step 2: Vérifier que le runner actuel ne couvre pas le navigateur**

Run: npm --prefix web run test:e2e -- --project=chromium

Expected: FAIL because le script existant lance Vitest, qui ne connaît pas l’option Playwright et ne découvre pas web/e2e.

- [x] **Step 3: Installer la dépendance verrouillée et ajouter le script**

Run from web:

~~~
npm install --save-dev --save-exact @playwright/test@1.63.0
~~~

Ajouter "test:e2e": "playwright test" dans web/package.json.

- [x] **Step 4: Ajouter la configuration Playwright**

Créer web/playwright.config.ts avec testDir './e2e', forbidOnly, deux retries en CI, workers à 1 en CI, baseURL http://127.0.0.1:4173, un webServer qui lance directement node node_modules/vite/bin/vite.js --configLoader runner --host 127.0.0.1 --port 4173 pour éviter un processus npm enfant orphelin sur Windows, et un seul projet Chromium basé sur devices Desktop Chrome.

- [x] **Step 5: Installer Chromium et faire passer le nominal**

Run from web:

~~~
npx playwright install chromium
npm run test:e2e
~~~

Expected: 1 test passed, 0 failures et aucune requête vers un domaine externe.

- [x] **Step 6: Committer le parcours nominal**

~~~
git add web/package.json web/package-lock.json web/playwright.config.ts web/e2e/dashboard.spec.ts
git commit -m "test: add browser dashboard e2e"
~~~

### Task 2: Couvrir le mode dégradé et la stabilité des fixtures

**Files:**
- Modify: web/e2e/dashboard.spec.ts

**Interfaces:**
- La fixture accepte qualityUnavailable et renvoie HTTP 503 uniquement pour la requête ErrorSummary.
- Le scénario prouve que le tableau conserve le véhicule 1234 malgré l’erreur qualité.

- [x] **Step 1: Écrire le scénario d’erreur**

Ajouter un test qui répond HTTP 503 pour ErrorSummary, puis vérifie:

~~~
await expect(page.getByRole('alert')).toContainText('Qualité indisponible pour le moment.')
await expect(page.getByRole('heading', { name: 'Véhicules en service' })).toBeVisible()
await expect(page.getByText('1234')).toBeVisible()
~~~

- [x] **Step 2: Exécuter les deux scénarios**

Run: npm --prefix web run test:e2e

Expected: 2 tests passed, 0 failures.

- [x] **Step 3: Vérifier le déterminisme des fixtures**

La fixture doit faire échouer toute requête non-GraphQL /query. Relancer deux fois:

~~~
npm --prefix web run test:e2e -- --repeat-each=2
~~~

Expected: 4 tests passés, 0 échec.

- [x] **Step 4: Committer la couverture dégradée**

~~~
git add web/e2e/dashboard.spec.ts
git commit -m "test: cover dashboard quality outage in browser"
~~~

### Task 3: Ajouter le job GitHub Actions et la documentation

**Files:**
- Modify: .github/workflows/ci.yml
- Modify: README.md
- Modify: docs/superpowers/plans/2026-09-12-jalon-6-browser-e2e.md

**Interfaces:**
- Le job CI s’appelle Browser E2E, utilise Node 24, installe Chromium et lance npm run test:e2e depuis web.
- Le rapport web/playwright-report est téléversé après la fin du job et ignore son absence.

- [x] **Step 1: Ajouter le job CI**

Ajouter un job qui checkout le dépôt, configure Node 24.x, exécute npm ci, npx playwright install --with-deps chromium, puis npm run test:e2e. Configurer working-directory: web, le cache npm et actions/upload-artifact@v4 sur web/playwright-report/ avec if-no-files-found: ignore et retention-days: 14.

- [x] **Step 2: Mettre à jour le README**

Dans « Vérifications locales », documenter:

~~~
npx playwright install chromium
npm --prefix web run test:e2e
~~~

Préciser que les E2E utilisent des fixtures locales et ne valident pas la disponibilité STM; le smoke test Compose reste le contrôle d’intégration.

- [x] **Step 3: Cocher le plan et valider le workflow**

Cocher toutes les étapes réalisées dans ce plan, puis lancer:

~~~
actionlint
git diff --check
~~~

Expected: aucune erreur actionlint ni whitespace error.

- [x] **Step 4: Committer CI et documentation**

~~~
git add .github/workflows/ci.yml README.md docs/superpowers/plans/2026-09-12-jalon-6-browser-e2e.md
git commit -m "ci: run browser e2e checks"
~~~

### Task 4: Validation complète et publication de la PR

**Files:**
- No source changes expected; validate the complete branch.

- [ ] **Step 1: Exécuter les vérifications locales complètes**

Run:

~~~
npm --prefix web run test -- --run
npm --prefix web run typecheck
npm --prefix web run build
npm --prefix web run test:e2e
pwsh -NoProfile -File scripts/check.ps1
pwsh -NoProfile -File scripts/smoke.ps1
~~~

Expected: chaque commande sort avec le code 0; le smoke test signale au moins un événement et un véhicule fixture.

- [ ] **Step 2: Vérifier sécurité et état Git**

Run:

~~~
git diff --check
rg -n "STM_API_KEY" web/dist web/playwright-report
git status --short --branch
~~~

La recherche de secret ne doit retourner aucune correspondance et l’arbre ne doit contenir que les changements prévus.

- [ ] **Step 3: Publier la branche et ouvrir la PR**

~~~
git push -u origin jalon-6
gh pr create --repo MinaSeliman1/Ponctuel --base jalon-5 --head jalon-6 --title "test: add browser e2e coverage"
~~~

La PR cible jalon-5, reste ouverte et ne doit pas être fusionnée automatiquement. Vérifier ensuite que tous les jobs CI, notamment Browser E2E, passent.
