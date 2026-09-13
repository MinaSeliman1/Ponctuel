# Jalon 21 — Focus clavier de la fiche carte

## Objectif

Permettre aux utilisateurs clavier de fermer la fiche d’un autobus avec Échap sans perdre leur position dans la carte.

## Comportement attendu

- Échap ferme la fiche de détail ouverte;
- le bouton de fermeture ferme aussi la fiche et restitue le focus au marqueur qui l’a ouverte;
- le marqueur reste sélectionnable par `Enter` et `Espace`;
- si la couche des positions est masquée, la sélection est supprimée sans tentative de focus sur un élément caché;
- la restauration de focus ne modifie ni les données, ni l’URL, ni les filtres du tableau.

## Hors périmètre

- comportement modal ou verrouillage global du focus;
- nouvelles données cartographiques;
- dépendance d’accessibilité ou service payant.

## Validation

- tests unitaires du retour de focus et de la fermeture Échap;
- test E2E Chromium du parcours clavier complet;
- tests frontend, typecheck, build, `check.ps1`, `smoke.ps1`, Docker/Helm et scan des secrets.
