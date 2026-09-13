# Jalon 13 — lien de filtres copiable

## Objectif

Permettre à une personne qui consulte le tableau des véhicules de copier en un
clic l’URL correspondant à la recherche et aux filtres actifs.

## Décisions

- Le bouton sera placé dans la barre d’actions du tableau, près de l’export CSV.
- Le lien copié sera une URL absolue afin d’être directement partageable hors
  du navigateur courant.
- La copie utilisera l’API Clipboard du navigateur et affichera un état
  accessible de succès ou d’échec.
- Les paramètres non liés aux filtres seront conservés dans le lien.
- Une modification de la recherche ou des filtres réinitialisera le message de
  copie, sans créer d’entrée supplémentaire dans l’historique.

## Hors périmètre

- Aucun stockage serveur, compte utilisateur ou service payant.
- Aucun raccourcisseur d’URL.
- Aucun changement au contrat GraphQL, à la pagination ou à l’export CSV.

## Critères d’acceptation

1. Le bouton copie une URL absolue contenant l’état courant des filtres.
2. Le succès et l’échec sont annoncés avec un statut lisible par lecteur
   d’écran.
3. Le bouton reste utilisable avec les filtres par défaut et après un reset.
4. Les tests unitaires, composants et Chromium couvrent le parcours.
