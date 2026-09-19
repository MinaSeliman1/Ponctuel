# Déploiement public gratuit : Render + Supabase

Ce profil publie le dashboard et l’API sur Render Free et conserve les données
dans Supabase Free. Il démarre par défaut en mode `fixture`, donc aucune clé STM
n’est nécessaire pour la première mise en ligne.

Le profil public remplace Redpanda par un bus mémoire limité au conteneur et
démarre quand même le matcher. Les données persistantes restent dans Supabase;
Compose reste le profil local qui démontre la variante distribuée avec Redpanda
et TimescaleDB.

## 1. Créer Supabase

1. Crée un projet Supabase sur le plan Free ($0).
2. Dans `Connect`, choisis le pooler partagé en mode session et copie la chaîne
   PostgreSQL. Remplace le mot de passe dans cette chaîne; garde `sslmode=require`.
3. Ne committe jamais cette chaîne : elle contient le mot de passe PostgreSQL.

Le démarrage Render exécute automatiquement les fichiers de `db/migrations` dans
Supabase. Le schéma fonctionne avec PostgreSQL standard; TimescaleDB reste une
optimisation uniquement lorsqu’elle est disponible localement.

## 2. Créer Render

1. Dans Render, ouvre `New > Blueprint` et sélectionne le dépôt public.
2. Render détecte `render.yaml` et crée `ponctuel-demo` sur le plan Free.
3. Dans les variables demandées, colle la valeur `DATABASE_URL` de Supabase.
4. Laisse `APP_ENV=fixture` pour la première mise en ligne, puis lance le déploiement.
5. Ouvre l’URL `https://ponctuel-demo.onrender.com` affichée par Render.

La première ouverture peut prendre environ une minute après une période
d’inactivité : Render Free met les services en veille après 15 minutes sans
trafic. Les données restent dans Supabase; le disque local Render est éphémère.

## 3. Activer les données STM réelles (optionnel)

Dans les variables d’environnement Render, ajoute la clé STM comme `STM_API_KEY`
et change `APP_ENV` en `stm`. La clé reste côté serveur et n’est jamais envoyée
au navigateur. Le service utilise `STM_API_KEY_HEADER=apikey` par défaut.

## Vérification

```powershell
Invoke-WebRequest https://<ton-service>.onrender.com/readyz
```

Une réponse `200` avec `ready` confirme que Render atteint Supabase et que l’API
est prête. Le dashboard est ensuite disponible sur la même URL.
