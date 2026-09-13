# Jalon 22 — résilience du rafraîchissement

Date : 2026-09-13  
Statut : approuvé pour implémentation progressive

## Objectif

Préserver l’expérience du dashboard lorsqu’une actualisation manuelle ou
automatique échoue après qu’une première collecte valide a déjà été affichée.
Une panne transitoire ne doit pas remplacer la carte, la table ou le panneau
de qualité par un état d’erreur vide.

## Décisions

- Séparer l’erreur de chargement initial (`error`) de l’échec non destructif
  d’une relance (`refreshError`).
- Conserver les valeurs existantes de `dashboard`, `vehicles` et
  `errorSummaries` pendant une relance; ne vider les résumés qu’au tout premier
  chargement.
- Afficher un avertissement accessible dans le panneau de statut, avec le rôle
  `alert`, sans masquer les données connues.
- Garder le bouton de relance actif après l’échec et le libeller « Réessayer ».
- Ignorer les annulations `AbortError`, afin qu’un démontage ou une course de
  requêtes ne soit pas présenté comme une panne réseau.
- Ne modifier aucun contrat GraphQL, secret, service payant ou comportement du
  mode STM.

## Contrat observable

1. Après une collecte réussie, si les requêtes de relance échouent, le véhicule
   connu reste visible dans la carte et la table.
2. Le panneau affiche que les dernières données valides sont conservées.
3. Le bouton reste activé et affiche « Réessayer ».
4. Une relance réussie supprime l’avertissement et remplace les données par la
   nouvelle collecte.
5. Un échec du premier chargement conserve le comportement d’erreur existant.

## Hors périmètre

- Pas de nouvelle stratégie de retry automatique ni de backoff.
- Pas de cache persistant, de stockage local ou de synchronisation serveur.
- Pas d’ajout de télémétrie ou de dépendance d’interface externe.

## Validation

- test Vue de conservation de la vue après trois réponses HTTP 503;
- test Playwright du bouton de relance et de l’avertissement accessible;
- suite frontend complète, typecheck et build;
- vérifications backend, Compose, Helm et CI sur la PR.
