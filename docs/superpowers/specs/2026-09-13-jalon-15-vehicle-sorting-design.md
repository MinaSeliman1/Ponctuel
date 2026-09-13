# Jalon 15 — Tri du tableau des véhicules

## Objectif

Permettre aux utilisateurs du dashboard de réordonner rapidement les véhicules
visibles afin de repérer une ligne, un véhicule précis ou les retards importants
sans modifier les données reçues ni perdre les filtres actifs.

## Périmètre

- Ajouter un sélecteur accessible de tri au-dessus du tableau.
- Proposer l’ordre d’arrivée, la ligne, l’identifiant du véhicule et le retard
  décroissant.
- Appliquer le tri après les filtres et avant la pagination.
- Garder l’ordre d’arrivée comme valeur par défaut et ne pas changer l’URL de
  partage des filtres.
- Couvrir le tri, les valeurs de retard nulles et la pagination par tests.

## Hors périmètre

- Aucun changement au backend, à l’API ou aux payloads STM.
- Aucun stockage persistant et aucun service payant.

## Critères d’acceptation

1. Le sélecteur porte un label explicite et est utilisable au clavier.
2. Le tri par retard place les plus grands retards en premier et les valeurs
   inconnues à la fin.
3. Le tri par ligne et par véhicule est stable et naturel pour les nombres.
4. Un changement de tri remet la pagination à la page 1.
5. Les filtres, les pastilles, le partage d’URL et l’export CSV continuent de
   fonctionner.
6. Les tests frontend, E2E, le typage, le build, le smoke test et la CI passent.
