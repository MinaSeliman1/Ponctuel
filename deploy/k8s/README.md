# Déploiement Kubernetes local

Le chart Helm de `deploy/k8s/chart/ponctuel` cible un cluster local ou k3s.
Il déploie les quatre services applicatifs, TimescaleDB et un broker Redpanda
de démonstration. Il ne fournit pas d’hébergement cloud, de sauvegarde ou de
SLA gratuit.

## Préparer les images et les migrations

Construis les images avec un tag versionné, puis charge-les dans le runtime du
cluster local ou publie-les dans un registre que tu contrôles :

```powershell
docker build -f Dockerfile.api -t ponctuel-api:0.3.0 .
docker build -f Dockerfile.ingester -t ponctuel-ingester:0.3.0 .
docker build -f Dockerfile.matcher -t ponctuel-matcher:0.3.0 .
docker build -f Dockerfile.web -t ponctuel-web:0.3.0 .
```

Les migrations ne sont pas copiées dans le chart pour éviter une deuxième
source de vérité. Crée leur ConfigMap depuis le checkout :

```powershell
kubectl create namespace ponctuel --dry-run=client -o yaml | kubectl apply -f -
kubectl -n ponctuel create configmap ponctuel-migrations `
  --from-file=db/migrations --dry-run=client -o yaml | kubectl apply -f -
```

## Lint et rendu

```powershell
helm lint deploy/k8s/chart/ponctuel
helm template ponctuel deploy/k8s/chart/ponctuel `
  -f deploy/k8s/values.local.example.yaml
```

Le rendu contient un Secret de développement avec un mot de passe de démonstration.
Remplace-le avec `--set database.password=...` ou un mécanisme de gestion de
secrets avant tout cluster partagé. Pour STM, passe `--set secrets.stmApiKey=...`
localement ou utilise un Secret géré par ton environnement; ne mets jamais la
clé dans GitHub, un ConfigMap ou l’image web.

## Installation locale

```powershell
helm upgrade --install ponctuel deploy/k8s/chart/ponctuel `
  --namespace ponctuel --create-namespace `
  -f deploy/k8s/values.local.example.yaml
kubectl -n ponctuel get pods,svc,pvc
kubectl -n ponctuel port-forward svc/ponctuel-ponctuel-web 3000:80
```

Pour supprimer uniquement cette installation :

```powershell
helm uninstall ponctuel --namespace ponctuel
kubectl delete namespace ponctuel
```

La persistence DB et Redpanda est volontairement locale au cluster. Une
exploitation sérieuse doit ajouter sauvegardes, rotation de secrets, Network-
Policies, stockage fiable et une stratégie de mise à jour validée.
