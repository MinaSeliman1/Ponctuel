#!/usr/bin/env bash
set -Eeuo pipefail

readonly BRANCH="${BRANCH:-jalon-1}"
readonly APP_DIR="${APP_DIR:-$HOME/Ponctuel}"
readonly COMPOSE_FILE="$APP_DIR/deploy/compose/docker-compose.yml"

if [[ ! -d "$APP_DIR/.git" ]]; then
  echo "Dépôt absent: exécute d'abord deploy/oci/bootstrap.sh." >&2
  exit 1
fi

git -C "$APP_DIR" pull --ff-only origin "$BRANCH"

if [[ ! -f "$APP_DIR/.env" ]]; then
  cp "$APP_DIR/.env.example" "$APP_DIR/.env"
fi

if ! grep -q '^WEB_PORT=' "$APP_DIR/.env"; then
  printf '\nWEB_PORT=80\n' >> "$APP_DIR/.env"
fi

sudo docker compose -p ponctuel -f "$COMPOSE_FILE" up --build --detach --remove-orphans

for attempt in $(seq 1 60); do
  if curl --fail --silent --show-error http://127.0.0.1/readyz >/dev/null; then
    echo "Ponctuel est prêt: $(git -C "$APP_DIR" rev-parse --short HEAD)"
    exit 0
  fi
  sleep 2
done

echo "Le dashboard public n'est pas devenu prêt après 120 secondes." >&2
sudo docker compose -p ponctuel -f "$COMPOSE_FILE" ps >&2 || true
exit 1
