# Jalon 16 — Partage de l’ordre de tri

## Objectif

Compléter le partage de la vue des véhicules : les paramètres de recherche et
de filtres sont déjà partageables, et ce jalon ajoute l’ordre de tri pour que le
destinataire voie la même liste dans le même ordre.

## Périmètre

- Ajouter un paramètre URL `sort` avec les valeurs `route`, `vehicle` et
  `delay`; l’ordre d’arrivée reste la valeur par défaut implicite.
- Restaurer le tri depuis l’URL au chargement.
- Synchroniser le tri dans l’URL sans créer d’entrée d’historique.
- Inclure le tri courant dans « Copier le lien ».
- Ignorer proprement les valeurs `sort` inconnues et préserver les paramètres
  URL sans rapport.
- Ajouter tests unitaires et E2E.

## Hors périmètre

- Aucun changement au backend ou aux données STM.
- Aucun stockage persistant et aucun service payant.

## Critères d’acceptation

1. `?sort=delay` ouvre directement le tableau trié par retard décroissant.
2. L’ordre d’arrivée ne génère pas de paramètre `sort` inutile.
3. Changer le tri met à jour l’URL avec `history.replaceState`.
4. Le lien copié contient recherche, filtres, tri et paramètres inconnus conservés.
5. Une valeur de tri inconnue revient à l’ordre d’arrivée sans casser la page.
6. Les tests frontend, E2E, le typage, le build, le smoke test et la CI passent.
