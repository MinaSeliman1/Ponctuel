# Jalon 8 — rafraîchissement explicite du dashboard

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Rendre le caractère vivant du dashboard démontrable : l’utilisateur peut
relancer la collecte depuis l’interface et l’application rafraîchit ensuite les
données périodiquement sans effacer les données déjà affichées.

## Décisions

- Le bouton « Actualiser » est placé dans le panneau de statut, près de la
  dernière collecte, avec un nom accessible et un état désactivé pendant la
  requête.
- Le premier chargement conserve le squelette actuel. Les rafraîchissements
  suivants utilisent un état `isRefreshing` distinct : carte, tableau et
  qualité restent visibles pendant la requête.
- Un rafraîchissement automatique de 30 secondes est lancé à l’affichage et
  annulé au démontage. Cette fréquence reste alignée avec le poller prévu par
  l’architecture et ne crée aucun nouvel appel STM côté navigateur : le
  navigateur appelle seulement l’API Go locale.
- Une erreur de rafraîchissement conserve les données précédentes et affiche
  l’état d’erreur déjà prévu; le bouton reste disponible pour réessayer.
- Aucun changement de schéma GraphQL, aucune dépendance et aucune clé STM ne
  sont nécessaires.

## Contrat observable

1. Après le chargement initial, le panneau affiche `Actualiser`.
2. Cliquer sur le bouton relance les trois requêtes dashboard, véhicules et
   qualité, puis laisse les résultats visibles.
3. Pendant la relance, le bouton porte `Actualisation…` et est désactivé.
4. Un rafraîchissement automatique est nettoyé lorsque l’application est
   démontée, afin d’éviter les requêtes orphelines.

## Hors périmètre

- Pas de WebSocket ni de service de notification externe.
- Pas de réglage utilisateur de l’intervalle dans cette étape.
- Pas de modification des limites ou de l’authentification GraphQL.

## Validation

- tests unitaires App/StatusPanel pour l’événement et le second chargement;
- test Playwright vérifiant que le bouton déclenche une nouvelle requête;
- typecheck, build, smoke Compose et CI complète.
