# Démonstration locale

Le Compose de ce dossier démarre une démonstration complète et gratuite :
TimescaleDB, l’ingester fixture, l’API GraphQL et le dashboard web.

```powershell
Copy-Item .env.example .env
docker compose -f deploy/compose/docker-compose.yml up --build
```

Ouvre ensuite [http://localhost:3000](http://localhost:3000). Les contrôles
API sont disponibles sur [http://localhost:8080/healthz](http://localhost:8080/healthz)
et [http://localhost:8080/readyz](http://localhost:8080/readyz).

Le mode par défaut est `fixture` : aucune requête réseau vers STM et aucune
clé ne sont nécessaires. Pour tester explicitement le mode STM, configure
`APP_ENV=stm` et `STM_API_KEY` uniquement dans le fichier `.env` local, puis
relance Compose. Ne mets jamais cette clé dans le dépôt, Vite ou l’image web.

Pour arrêter et nettoyer uniquement le projet Compose Ponctuel :

```powershell
docker compose -f deploy/compose/docker-compose.yml down --volumes --remove-orphans
```

