## Résumé

Décrire le changement et le problème résolu.

## Vérifications

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `npm --prefix web run test -- --run`
- [ ] `npm --prefix web run typecheck`
- [ ] `npm --prefix web run build`
- [ ] `docker compose -f deploy/compose/docker-compose.yml config`
- [ ] Smoke test fixture exécuté si le changement touche Compose ou l’ingestion
- [ ] `git diff --check`
- [ ] Documentation mise à jour si nécessaire
- [ ] Aucun secret, payload réel ou donnée personnelle ajouté
- [ ] Le changement reste compatible avec la démonstration gratuite

## Données et sécurité

- [ ] Les variables sensibles restent dans `.env` ou les secrets CI
- [ ] Les limites et l’attribution STM sont toujours respectées

## Risques et compatibilité

Décrire les migrations, changements de contrat, risques et plan de retour en
arrière. Indiquer `N/A` lorsqu’il n’y en a pas.
