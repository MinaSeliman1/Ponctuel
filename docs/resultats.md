# Résultats et méthode

## Ce qui est mesuré

Ponctuel relie une arrivée observée à toutes les prédictions déjà persistées
pour le même trajet, la même date de service et le même arrêt. Pour chaque
liaison :

```text
erreur_secondes = observed_at - predicted_at
```

Une erreur négative signifie que le véhicule est arrivé avant la prédiction;
une erreur positive signifie qu’il est arrivé après. L’horizon est calculé
entre `recorded_at` et `predicted_at`, puis borné à 0–3 600 secondes. Une
arrivée est comptée « à l’heure » dans la fenêtre asymétrique `[-60, 300]`
secondes. Ce choix est visible dans la migration et reste modifiable sans
changer le contrat GraphQL.

L’API expose les agrégats par ligne et horizon avec :

```graphql
{ errorSummary(limit: 100) {
    routeId horizonSeconds sampleCount meanErrorSeconds onTimeRate
} }
```

`sampleCount` accompagne chaque moyenne et chaque taux afin d’éviter de lire
un pourcentage sans connaître sa base statistique.

## Affichage dans le dashboard

Le panneau « Qualité des prédictions » consomme ces résumés et les regroupe par
`horizonSeconds`. Les lignes d’un même horizon sont combinées avec une moyenne
pondérée :

```text
erreur_horizon = Σ(meanErrorSeconds × sampleCount) / Σ(sampleCount)
taux_horizon   = Σ(onTimeRate × sampleCount) / Σ(sampleCount)
```

Le graphique SVG est accompagné d’un tableau HTML qui reprend l’horizon,
l’erreur moyenne, le taux « à l’heure » et le nombre d’observations. Lorsque
l’API est indisponible, le panneau affiche son propre état d’erreur et laisse
la carte et la liste des véhicules fonctionner. Lorsqu’il n’y a encore aucune
observation correspondante, il affiche un état vide explicite.

## Predictor

Le service `services/predictor` est volontairement indépendant de PostgreSQL,
Redpanda et de la clé STM. Il reçoit des observations tabulaires déjà
préparées et prédit une correction d’erreur en secondes à partir de la ligne,
de l’horizon, de l’heure et du délai courant. Le modèle est un
`GradientBoostingRegressor` déterministe (`random_state=42`).

La validation est temporelle : les observations sont triées par
`recorded_at`, les 20 % les plus récentes sont réservées au test et aucune
ligne future ne participe à l’entraînement. La baseline comparable prédit
toujours une correction de zéro. Les métriques publiées doivent donc toujours
indiquer le nombre de lignes de test, le MAE de la baseline et le MAE du
modèle.

L’image `Dockerfile.predictor` démarre même sans modèle afin que le dépôt reste
exécutable sans artefact binaire versionné. Dans ce cas `/healthz` indique
`waiting_for_model` et `/predict` répond `503`. Un pipeline d’entraînement peut
monter un artefact Joblib au chemin `PREDICTOR_MODEL_PATH`; l’image ne lit
aucun secret STM.

## Limites honnêtes

Les fixtures du dépôt servent à tester les contrats, l’idempotence et le
chemin de démonstration. Elles ne sont pas un échantillon STM représentatif et
aucun gain ML ne doit être annoncé à partir d’elles. Un résultat réel devra
documenter la période, la version des données, le nombre d’arrivées valides,
la distribution des horizons, la baseline et la séparation temporelle avant
d’être ajouté ici.
