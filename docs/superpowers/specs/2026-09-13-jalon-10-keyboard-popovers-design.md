# Jalon 10 — panneaux flottants accessibles au clavier

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Rendre les panneaux Préférences et Filtres cohérents avec la promesse
d’accessibilité du dashboard : une personne qui navigue au clavier doit
pouvoir ouvrir un panneau, commencer immédiatement à l’utiliser, le fermer
avec Échap et retrouver le bouton qui l’a ouvert.

## Décisions

- L’ouverture place le focus sur le premier contrôle utile du panneau.
- La touche Échap ferme le panneau actif.
- La fermeture par Échap ou par le bouton Fermer rend le focus au déclencheur.
- Les deux panneaux restent des popovers non modaux : le contenu principal
  n’est pas masqué et aucune dépendance externe n’est ajoutée.
- Les contrats GraphQL, les données STM et l’actualisation restent inchangés.

## Contrat observable

1. Ouvrir Préférences avec le bouton donne le focus à la case
   « Actualisation automatique ».
2. Échap ferme Préférences et redonne le focus au bouton Préférences.
3. Ouvrir Filtres donne le focus au sélecteur de ligne.
4. Échap ferme Filtres et redonne le focus au bouton Filtres.

## Hors périmètre

- Pas de focus trap modal.
- Pas de nouvelle librairie d’accessibilité.
- Pas de persistance des filtres ou préférences.

## Validation

- tests Vitest du focus et de la fermeture clavier;
- test Playwright du parcours clavier des deux popovers;
- typecheck, build, smoke Compose et CI complète.
