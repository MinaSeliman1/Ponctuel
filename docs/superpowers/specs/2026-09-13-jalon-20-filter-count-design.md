# Jalon 20 — Compteur de filtres cohérent

## Objectif

Faire correspondre le compteur du bouton `Filtres actifs` avec le résumé visible et les critères réellement appliqués au tableau.

## Comportement attendu

- une recherche non vide compte comme un filtre actif;
- la ligne et l’état de retard continuent d’être comptés;
- le compteur reste indépendant du tri, qui est un ordre d’affichage et non un filtre;
- le libellé accessible du bouton reflète exactement le nombre de critères actifs;
- la pastille de recherche et le compteur restent synchronisés quand la recherche est modifiée ou effacée.

## Hors périmètre

- modification du format des paramètres URL;
- changement du fonctionnement du tri ou de l’export CSV;
- ajout de dépendance ou de service payant.

## Validation

- test unitaire du compteur pour une recherche seule;
- test E2E Chromium de synchronisation recherche/pastille/compteur;
- tests frontend, typecheck, build, `check.ps1`, `smoke.ps1`, Docker/Helm et scan des secrets.
