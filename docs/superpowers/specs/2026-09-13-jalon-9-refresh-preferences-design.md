# Jalon 9 — préférences de rafraîchissement

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Rendre le bouton « Préférences » du dashboard fonctionnel et transparent :
l’utilisateur peut suspendre ou réactiver le rafraîchissement automatique,
tandis que le bouton « Actualiser » reste toujours disponible.

## Décisions

- Le bouton ouvre un petit dialogue accessible dans la barre supérieure, sans
  dépendance externe ni navigation.
- Une seule préférence est exposée pour l’instant : `Actualisation
  automatique`, activée par défaut.
- La désactivation annule immédiatement le timer existant; la réactivation
  crée un nouveau timer de 30 secondes.
- La préférence est volontairement en mémoire seulement et revient à son état
  par défaut au rechargement. Il n’y a pas de stockage local de données ni de
  secret.
- Le rafraîchissement manuel du jalon 8 n’est jamais désactivé.

## Contrat observable

1. Le bouton Préférences expose `aria-expanded` et ouvre un dialogue nommé
   « Préférences ».
2. La case « Actualisation automatique » est cochée par défaut.
3. La décocher suspend le timer et affiche clairement l’état désactivé.
4. Le bouton Actualiser reste utilisable et relance les trois requêtes locales.

## Hors périmètre

- Pas de persistance entre sessions.
- Pas de choix d’intervalle arbitraire.
- Pas de modification des contrats GraphQL ou du mode STM.

## Validation

- tests unitaires du toggle et du rafraîchissement manuel;
- test Playwright du dialogue, de la désactivation et de la relance manuelle;
- typecheck, build, smoke Compose et CI complète.
