#!/usr/bin/env bash
set -Eeuo pipefail

# Bootstrap one OCI Always Free Ubuntu VM and start the public demonstration.
# The script intentionally keeps the STM key out of the repository.

readonly REPOSITORY_URL="${REPOSITORY_URL:-https://github.com/MinaSeliman1/Ponctuel.git}"
readonly BRANCH="${BRANCH:-jalon-1}"
readonly APP_DIR="${APP_DIR:-$HOME/Ponctuel}"

if [[ "$(id -u)" -eq 0 ]]; then
  echo "Lance ce script avec l'utilisateur Ubuntu, pas directement avec root." >&2
  exit 1
fi

if ! command -v git >/dev/null 2>&1 || \
   ! command -v curl >/dev/null 2>&1 || \
   ! command -v docker >/dev/null 2>&1 || \
   ! sudo docker compose version >/dev/null 2>&1; then
  sudo apt-get update
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates \
    curl \
    docker.io \
    docker-compose-v2 \
    git
  sudo systemctl enable --now docker
fi

if ! sudo docker compose version >/dev/null 2>&1; then
  echo "Docker Compose v2 est requis sur la VM." >&2
  exit 1
fi

sudo usermod -aG docker "$USER" || true
mkdir -p "$(dirname "$APP_DIR")"

if [[ ! -d "$APP_DIR/.git" ]]; then
  git clone --branch "$BRANCH" --single-branch "$REPOSITORY_URL" "$APP_DIR"
else
  git -C "$APP_DIR" pull --ff-only origin "$BRANCH"
fi

if [[ ! -f "$APP_DIR/.env" ]]; then
  cp "$APP_DIR/.env.example" "$APP_DIR/.env"
fi

# The public entry point is the web container on port 80. API and matcher
# remain behind the OCI firewall and are only used by the Compose network.
if grep -q '^WEB_PORT=' "$APP_DIR/.env"; then
  sed -i 's/^WEB_PORT=.*/WEB_PORT=80/' "$APP_DIR/.env"
else
  printf '\nWEB_PORT=80\n' >> "$APP_DIR/.env"
fi

bash "$APP_DIR/deploy/oci/deploy.sh"

echo
echo "Déploiement terminé. Ouvre http://<IP-PUBLIQUE-DE-LA-VM>"
echo "Pour activer les données STM réelles, configure APP_ENV et STM_API_KEY uniquement dans $APP_DIR/.env."
