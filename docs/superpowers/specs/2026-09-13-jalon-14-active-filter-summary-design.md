# Jalon 14 — Résumé des filtres actifs

## Objectif

Rendre l’état filtré immédiatement compréhensible dans le tableau des véhicules,
sans obliger l’utilisateur à rouvrir le panneau Filtres. Chaque critère actif est
présenté sous forme de pastille accessible et peut être retiré individuellement.

## Périmètre

- Afficher les pastilles pour la recherche, la ligne et l’état de retard.
- Permettre de supprimer un critère avec un bouton nommé et accessible.
- Fournir une action visible pour réinitialiser tous les critères actifs.
- Conserver la synchronisation URL, la pagination et le bouton « Copier le lien ».
- Ajouter des tests unitaires et E2E ciblant le comportement utilisateur.

## Hors périmètre

- Aucun changement à l’API, à la persistance ou au contrat de données.
- Aucun service payant, aucune dépendance externe et aucun changement de design
  global.

## Critères d’acceptation

1. Avec au moins un critère actif, un résumé « Filtres actifs » est visible sous
   la barre d’outils.
2. Chaque pastille expose une action de suppression avec un nom explicite.
3. Retirer une pastille met à jour immédiatement le tableau et les paramètres
   `q`, `route` et `delay` de l’URL.
4. « Réinitialiser tout » vide les critères, remet la pagination à la première
   page et supprime les paramètres de filtre de l’URL.
5. Le résumé n’est pas rendu lorsque tous les critères sont neutres.
6. Les tests frontend, le typage, le build et le parcours E2E passent.
