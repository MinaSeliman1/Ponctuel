# Jalon 19 — Contrôles de carte cohérents

## Objectif

Aligner les contrôles visibles de la carte sur les données réellement disponibles dans la démonstration et expliquer la signification des couleurs des véhicules.

## Comportement attendu

- le contrôle `Réseau routier` masque et réaffiche les rues schématiques;
- le contrôle `Positions des autobus` masque et réaffiche les marqueurs;
- les couches non alimentées par le contrat actuel (`Arrêts`, `Zones de service`) ne sont plus présentées comme disponibles;
- la carte expose une légende accessible pour les véhicules à l’heure, légèrement en retard et fortement en retard;
- masquer les positions ferme une fiche de véhicule ouverte sans laisser d’état incohérent;
- les contrôles restent utilisables au clavier et ne requièrent aucun service externe ou payant.

## Hors périmètre

- ajout de données d’arrêts ou de zones non présentes dans l’API;
- intégration d’une bibliothèque cartographique ou de tuiles externes;
- modification du calcul de retard ou du contrat GraphQL.

## Validation

- tests unitaires des deux contrôles, de la légende et du nettoyage de sélection;
- test E2E Chromium de masquage/réaffichage des couches;
- tests frontend, typecheck, build, `check.ps1`, `smoke.ps1`, Docker/Helm et scan des secrets.
