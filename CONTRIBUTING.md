# Contribuer

Ponctuel est développé en français, en mode fixture-first. Une contribution
doit rester reproductible sans clé STM ni service payant.

## Développement local

```powershell
Copy-Item .env.example .env
docker compose -f deploy/compose/docker-compose.yml up --build
```

Pour les tests ciblés :

```powershell
go test ./...
go vet ./...
npm --prefix web ci
npm --prefix web run test -- --run
npm --prefix web run typecheck
npm --prefix web run build
```

Les changements qui touchent l’ingestion, l’API ou Compose doivent inclure
une vérification fixture. Les changements web doivent conserver les états
chargement, erreur, vide et données vieillissantes, ainsi que les labels
accessibles.

## Pull requests

Explique le problème, la décision technique et les commandes exécutées. Ne
publie jamais `STM_API_KEY`, un fichier `.env`, une réponse réelle complète ou
des données personnelles dans une issue, une PR ou un log.

