# Jalon 11 — export CSV des véhicules filtrés

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Permettre à l’utilisateur d’emporter les données visibles du tableau des
véhicules pour une analyse locale ou une démonstration de portfolio. L’export
doit respecter la recherche et les filtres actifs, sans nouvel appel réseau.

## Décisions

- Un bouton « Exporter CSV » est placé dans la barre d’actions du tableau.
- L’export contient tous les véhicules correspondant aux critères actifs, pas
  uniquement la page courante.
- Les colonnes utilisent des noms stables et des valeurs brutes utiles à
  l’analyse : ligne, véhicule, trajet, latitude, longitude, position UTC et
  retard en secondes.
- Le fichier est généré dans le navigateur avec `Blob` et un téléchargement
  local; aucune donnée n’est envoyée à un service externe.
- Le bouton est désactivé pendant le chargement, en cas d’erreur ou lorsqu’il
  n’y a aucune ligne exportable.

## Contrat observable

1. Sans filtre, le bouton exporte tous les véhicules chargés.
2. Avec une recherche ou un filtre, seules les lignes correspondantes sont
   exportées.
3. Le CSV échappe les guillemets, virgules et retours de ligne correctement.
4. Le nom du fichier commence par `ponctuel-vehicules-` et finit par `.csv`.

## Hors périmètre

- Pas d’export du graphique ou de l’historique.
- Pas de stockage serveur ni de compte utilisateur.
- Pas de nouvelle dépendance.

## Validation

- tests unitaires de sérialisation CSV et d’état du bouton;
- test Playwright du téléchargement après filtrage;
- typecheck, build, smoke Compose et CI complète.
