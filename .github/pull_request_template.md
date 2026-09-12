## Résumé

<!-- Décris le changement et son impact utilisateur ou opérationnel. -->

## Vérifications

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `npm --prefix web run test -- --run`
- [ ] `npm --prefix web run typecheck`
- [ ] `npm --prefix web run build`
- [ ] `docker compose -f deploy/compose/docker-compose.yml config`
- [ ] Smoke test fixture exécuté si le changement touche Compose ou l’ingestion

## Données et sécurité

- [ ] Aucun secret, payload réel inutile ou clé STM ajouté
- [ ] Les variables sensibles restent dans `.env` ou les secrets CI
- [ ] Les limites et l’attribution STM sont toujours respectées

## Notes pour la revue

<!-- Ajoute les limites connues, captures utiles ou décisions d’architecture. -->

