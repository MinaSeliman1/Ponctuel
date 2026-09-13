# Ponctuel — conception du jalon 6 E2E navigateur

## Objectif

Ajouter une vérification navigateur reproductible du dashboard Ponctuel afin
de valider le parcours visible par un utilisateur sans dépendre de l’API STM,
d’une clé secrète, d’un service payant ou d’un environnement Docker complet.

## Décision

Le jalon utilise `@playwright/test` avec Chromium et un serveur Vite local.
Chaque test intercepte les requêtes `POST /query` et renvoie des réponses
GraphQL fixture déterministes pour le dashboard, les véhicules et la qualité
des prédictions. La page est donc testée dans un vrai navigateur, tandis que
les données et les pannes restent contrôlées par le test.

Cette approche est préférable à un test Compose complet à chaque scénario:
elle vérifie le rendu et les interactions réelles avec un coût CI faible et
sans flakiness provenant de TimescaleDB, Redpanda ou du réseau STM. Le smoke
test Compose déjà présent reste responsable de l’intégration backend.

## Périmètre

Le jalon ajoute:

- une configuration Playwright Chromium avec `webServer` Vite;
- un scénario nominal qui vérifie le statut réseau, la carte, la liste des
  véhicules et le graphique de qualité;
- un scénario de panne `errorSummary` qui vérifie que le tableau reste
  utilisable;
- un script `npm run test:e2e` isolé des tests Vitest;
- un job CI frontend E2E avec installation du navigateur Chromium;
- la documentation de la commande et de la stratégie fixture-first.

Le jalon n’ajoute pas de navigateur dans l’image de production, de test
d’API STM réelle, de données personnelles, de snapshots visuels fragiles ou
de dépendance cloud.

## Contrat de test

Le routeur fixture inspecte le champ GraphQL demandé et retourne uniquement
les données minimales nécessaires. Les réponses nominales contiennent:

- un dashboard en mode `STM` avec 24 événements;
- un autobus `1234` sur la ligne `51`;
- deux résumés de qualité au même horizon pour vérifier que le rendu reçoit
  le résultat pondéré `+30 s` déjà couvert par Vitest.

Les scénarios vérifient les rôles et noms accessibles plutôt que des sélecteurs
CSS internes. Le scénario d’erreur renvoie HTTP 503 uniquement pour
`errorSummary`, puis vérifie le message de qualité et la présence de la ligne
`1234`.

## CI et exécution locale

Le script est exécuté depuis `web`:

```powershell
npm ci
npx playwright install --with-deps chromium
npm run test:e2e
```

Le workflow garde les jobs Vitest/typecheck/build existants et ajoute un job
E2E séparé. Le job utilise Node 24, installe Chromium, lance Playwright et
conserve le rapport HTML uniquement en cas d’échec. Aucun secret n’est requis.

## Critères d’acceptation

1. `npm run test:e2e` passe avec Chromium sur un clone propre après
   installation des dépendances et du navigateur.
2. Le test nominal prouve la présence des sections principales et du graphique
   accessible dans un navigateur réel.
3. Le test d’erreur prouve que la qualité peut échouer sans cacher les
   véhicules.
4. Le job CI E2E est indépendant des secrets STM et ne contacte aucun domaine
   externe pendant les scénarios.
5. La documentation distingue explicitement fixtures de test et mesures STM.
