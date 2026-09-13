# Jalon 18 — Détails accessibles depuis la carte

## Objectif

Rendre chaque marqueur d’autobus de la carte interactive et accessible afin qu’un utilisateur puisse consulter rapidement les informations du véhicule sans passer par le tableau.

## Comportement attendu

- chaque marqueur rendu reçoit le rôle `button`, un nom accessible et un `tabindex` clavier;
- un clic, `Enter` ou `Espace` sélectionne le véhicule;
- le marqueur sélectionné expose `aria-pressed="true"` et un style visuel distinct;
- une fiche accessible présente l’identifiant du véhicule, la ligne, le trajet, le retard, la position et l’heure de collecte;
- la fiche peut être fermée et disparaît si le véhicule sélectionné n’existe plus dans les données reçues;
- les données de la carte restent en lecture seule : aucune nouvelle API ou dépendance payante n’est ajoutée.

## Hors périmètre

- ajout d’un fournisseur de cartes externe;
- modification des données STM ou du contrat GraphQL;
- synchronisation de la sélection avec l’URL.

## Validation

- tests unitaires du composant `VehicleMap` pour sélection, clavier, fiche et disparition;
- test E2E Chromium pour ouverture depuis le marqueur et fermeture de la fiche;
- tests frontend, typecheck, build, `check.ps1`, `smoke.ps1`, validation Docker/Helm et analyse des secrets.
