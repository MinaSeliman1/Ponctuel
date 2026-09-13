# Jalon 7 — contrôles réels du tableau des véhicules

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Remplacer les contrôles visuels non fonctionnels du tableau des véhicules par
des interactions vérifiables : filtrage par ligne et état de retard, recherche
existante conservée, pagination réelle et états accessibles.

Cette étape réalise la promesse « tableau filtrable par ligne » du cahier
technique et améliore la crédibilité de la démonstration publique. Elle ne
modifie pas l’API GraphQL : les données déjà chargées sont filtrées localement
pour éviter une nouvelle requête et ne pas augmenter la charge STM.

## Décisions

- Utiliser uniquement Vue 3 et les éléments HTML natifs (`button`, `select`,
  `input`, `nav`) : aucune dépendance supplémentaire.
- Le bouton « Filtres » ouvre un panneau identifié par `aria-controls` et
  `aria-expanded`. Les filtres sont appliqués immédiatement.
- Le filtre de ligne propose les valeurs distinctes présentes dans la réponse,
  avec une option « Toutes les lignes ».
- Le filtre de retard propose « Tous », « À l’heure ou léger retard »
  (`delaySeconds <= 120` ou valeur inconnue) et « Retard important »
  (`delaySeconds > 120`). La règle reste alignée sur les classes visuelles du
  tableau.
- La pagination affiche 10 véhicules par page, remet la page à 1 lorsqu’un
  filtre ou une recherche change et expose une page précédente/suivante
  désactivée aux limites.
- Le compteur du bouton indique uniquement le nombre de filtres avancés actifs;
  la recherche est déjà visible dans son propre champ.
- Les fixtures E2E restent locales et déterministes; aucune requête STM ou clé
  n’est ajoutée.

## Contrat observable

Le tableau doit permettre de :

1. ouvrir le panneau avec le bouton `Filtres`;
2. choisir une ligne et constater que seuls les véhicules de cette ligne sont
   affichés;
3. choisir `Retard important` et masquer les véhicules à l’heure;
4. naviguer entre les pages quand plus de 10 véhicules sont présents;
5. réinitialiser les filtres et retrouver l’ensemble des véhicules.

Les assertions privilégient les rôles, labels et noms accessibles. Un état
vide explique si aucune donnée ne correspond à la recherche ou aux filtres.

## Hors périmètre

- Pas de filtrage serveur ni de changement GraphQL.
- Pas de carte interactive externe ni de tuiles payantes.
- Pas de persistance des préférences entre deux sessions.
- Pas de modification du bouton « Préférences » global, qui sera traité dans
  une étape UX séparée si nécessaire.

## Validation

- tests Vitest du composant pour recherche, filtres et pagination;
- parcours Playwright nominal étendu au panneau de filtres et à la pagination;
- typecheck, build, suite locale et CI complète;
- scan de secrets inchangé.
