# Plan — Jalon 17 : navigation navigateur

## Tâche 1 — Définir la synchronisation

- [x] Choisir `popstate` comme événement d’entrée de navigation.
- [x] Conserver `replaceState` pour les changements de contrôles.

## Tâche 2 — Implémenter le cycle de vie

- [x] Ajouter la restauration d’état depuis l’URL.
- [x] Enregistrer et retirer le listener dans le cycle de vie Vue.
- [x] Réinitialiser la pagination et les retours d’action.

## Tâche 3 — Tester

- [x] Ajouter les tests unitaires de navigation et de nettoyage.
- [x] Ajouter un parcours E2E de retour/avance.

## Tâche 4 — Vérifier et publier

- [x] Mettre à jour le README.
- [x] Exécuter la validation locale complète et le smoke test.
- [ ] Publier le PR sans le fusionner automatiquement.
