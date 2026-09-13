# Jalon 12 — filtres partageables via l’URL

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Permettre de partager un état d’analyse du tableau des véhicules. La recherche
et les filtres actifs doivent être encodés dans l’URL, restaurés à l’ouverture
de la page et mis à jour sans rechargement.

## Décisions

- Les paramètres sont `q` pour la recherche, `route` pour la ligne et `delay`
  pour l’état de retard (`all`, `on-time` ou `late`).
- Les paramètres absents reprennent les valeurs par défaut.
- `history.replaceState` est utilisé : le bouton Retour du navigateur n’est
  pas pollué par chaque frappe.
- Les valeurs inconnues sont ignorées et retombent sur les valeurs sûres.
- Aucun véhicule, secret ou appel réseau n’est ajouté à l’URL.

## Contrat observable

1. Une URL contenant `?route=80&delay=late` ouvre le tableau avec ces filtres.
2. Une modification de la recherche, de la ligne ou du retard met à jour
   l’URL sans recharger la page.
3. Réinitialiser les filtres supprime les paramètres associés.
4. Copier l’URL et la rouvrir restitue le même état de tableau.

## Hors périmètre

- Pas de synchronisation entre plusieurs onglets.
- Pas de stockage local ou serveur.
- Pas de changement GraphQL.

## Validation

- tests unitaires de lecture/écriture des paramètres et du tableau;
- test Playwright d’ouverture avec une URL filtrée et de partage de l’état;
- typecheck, build, smoke Compose et CI complète.
