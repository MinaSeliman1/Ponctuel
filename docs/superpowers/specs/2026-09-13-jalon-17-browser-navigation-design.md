# Jalon 17 — Navigation navigateur et état partageable

## Objectif

Éviter qu’une navigation directe, un retour ou une avance du navigateur laisse
le tableau des véhicules différent de l’URL affichée. Les filtres et le tri
doivent être une source d’état réactive, y compris lorsque l’URL change depuis
une action de navigation du navigateur.

## Périmètre

- Écouter `popstate` lorsque le tableau est monté.
- Restaurer recherche, ligne, retard et tri depuis l’URL lors de l’événement.
- Recalculer la page et effacer le statut de copie après navigation.
- Retirer l’écouteur au démontage pour éviter les fuites et les doublons.
- Ajouter tests unitaires et E2E ciblant retour/avance.

## Hors périmètre

- Ne pas remplacer `replaceState` par `pushState` : les changements de filtre
  instantanés ne doivent pas remplir l’historique à chaque frappe.
- Aucun changement backend, aucune dépendance et aucun service payant.

## Critères d’acceptation

1. Un événement `popstate` met à jour les champs visibles et les lignes filtrées.
2. Le tri restauré depuis l’URL est appliqué immédiatement.
3. La pagination revient à la page 1 et le statut « lien copié » est réinitialisé.
4. Le listener est retiré au démontage du composant.
5. Les URLs inconnues restent sûres et les paramètres sans rapport sont conservés.
6. Les tests frontend, E2E, le typage, le build, le smoke test et la CI passent.
