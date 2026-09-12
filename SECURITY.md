# Sécurité

## Règles de base

- `STM_API_KEY` est uniquement une variable d’environnement locale ou un
  secret GitHub Actions; elle n’est jamais injectée dans Vite ou l’image web.
- Le mode fixture est le mode par défaut de la démonstration et de la CI.
- Les URLs STM, les payloads et les erreurs exposées au navigateur sont
  contrôlés; les clés ne doivent pas apparaître dans les logs.
- `.env`, `data/` et les artefacts de build sont ignorés par Git.

## Signaler une vulnérabilité

N’ouvre pas d’issue publique avec un secret ou une preuve exploitable. Utilise
un avis de sécurité privé GitHub du dépôt, si disponible, et fournis une
description minimale, les versions concernées et les étapes de reproduction.
Ne transmets aucune clé STM dans le rapport; révoque-la immédiatement auprès
du fournisseur si elle a été exposée.

## Limites actuelles

Le jalon 1 est une démonstration locale et un socle de portfolio. Il ne
comprend pas encore l’authentification utilisateur, la haute disponibilité,
la rotation automatisée des secrets, la rétention opérationnelle ou une
garantie de SLA. Une mise en production devra ajouter ces contrôles et une
revue de confidentialité des données de transport.

