#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="${DEPLOY_HOST:-82.156.54.232}"
REMOTE_USER="${DEPLOY_USER:-ubuntu}"
REMOTE_PROJECT_DIR="${REMOTE_PROJECT_DIR:-/home/ubuntu/campaign-lottery-platform}"
NGINX_SITE_PATH="${NGINX_SITE_PATH:-/etc/nginx/sites-available/gaokao-api}"
SSH_PASSWORD="${DEPLOY_PASSWORD:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
CORS_ALLOW_ORIGIN="${CORS_ALLOW_ORIGIN:-*}"

if [[ -z "$SSH_PASSWORD" ]]; then
  echo "DEPLOY_PASSWORD is required" >&2
  exit 1
fi

if [[ -z "$ADMIN_PASSWORD" ]]; then
  echo "ADMIN_PASSWORD is required" >&2
  exit 1
fi

if [[ -z "$MYSQL_PASSWORD" ]]; then
  echo "MYSQL_PASSWORD is required" >&2
  exit 1
fi

command -v sshpass >/dev/null 2>&1 || {
  echo "sshpass is required" >&2
  exit 1
}

rsync_rsh="sshpass -p '$SSH_PASSWORD' ssh -o StrictHostKeyChecking=no"
ssh_cmd=(sshpass -p "$SSH_PASSWORD" ssh -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST")
scp_cmd=(sshpass -p "$SSH_PASSWORD" scp -o StrictHostKeyChecking=no)

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

cp "$ROOT_DIR/deploy/pm2/ecosystem.config.cjs" "$tmp_dir/ecosystem.config.cjs"
cp "$ROOT_DIR/deploy/nginx/gaokao-api.conf" "$tmp_dir/gaokao-api.conf"

python3 - "$tmp_dir/ecosystem.config.cjs" "$ADMIN_PASSWORD" "$MYSQL_PASSWORD" "$CORS_ALLOW_ORIGIN" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
admin_password = sys.argv[2]
mysql_password = sys.argv[3]
cors_allow_origin = sys.argv[4]

text = path.read_text()
text = text.replace("ADMIN_PASSWORD: 'change-me'", f"ADMIN_PASSWORD: '{admin_password}'")
text = text.replace("MYSQL_PASSWORD: 'change-me'", f"MYSQL_PASSWORD: '{mysql_password}'")
text = text.replace("CORS_ALLOW_ORIGIN: '*'", f"CORS_ALLOW_ORIGIN: '{cors_allow_origin}'")
path.write_text(text)
PY

echo "[1/5] Sync source code to $REMOTE_USER@$REMOTE_HOST"
RSYNC_RSH="$rsync_rsh" rsync -az --delete \
  --exclude node_modules \
  --exclude .next \
  --exclude .git \
  --exclude '.env' \
  --exclude '.env.local' \
  "$ROOT_DIR/payment-module/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PROJECT_DIR/payment-module/"
RSYNC_RSH="$rsync_rsh" rsync -az --delete \
  --exclude node_modules \
  --exclude .next \
  --exclude .git \
  --exclude '.env' \
  --exclude '.env.local' \
  "$ROOT_DIR/backend-server/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PROJECT_DIR/backend-server/"
RSYNC_RSH="$rsync_rsh" rsync -az --delete \
  --exclude node_modules \
  --exclude .next \
  --exclude .git \
  --exclude '.env' \
  --exclude '.env.local' \
  "$ROOT_DIR/front-page/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PROJECT_DIR/front-page/"
RSYNC_RSH="$rsync_rsh" rsync -az "$ROOT_DIR/sql/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PROJECT_DIR/sql/"

echo "[2/5] Upload deploy templates"
"${scp_cmd[@]}" "$tmp_dir/ecosystem.config.cjs" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PROJECT_DIR/ecosystem.config.cjs"
"${scp_cmd[@]}" "$tmp_dir/gaokao-api.conf" "$REMOTE_USER@$REMOTE_HOST:/tmp/campaign-lottery-gaokao-api.conf"

echo "[3/5] Install backend dependencies and run migrations"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/backend-server' && npm install --no-fund --no-audit && npm run migrate"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/payment-module' && rm -rf node_modules dist && ln -s ../backend-server/node_modules node_modules && ../backend-server/node_modules/.bin/tsc -p tsconfig.build.json"
"${ssh_cmd[@]}" "rm -rf '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery/payment-module' && mkdir -p '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery' && cp -R '$REMOTE_PROJECT_DIR/payment-module' '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery/payment-module' && rm -rf '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery/payment-module/node_modules'"

echo "[4/5] Build frontend and backend"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/backend-server' && npm run build"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/front-page' && npm install --no-fund --no-audit && npm run build"

echo "[5/5] Replace PM2/nginx config and reload services"
"${ssh_cmd[@]}" "set -e; \
  cp '$REMOTE_PROJECT_DIR/ecosystem.config.cjs' '$REMOTE_PROJECT_DIR/ecosystem.config.cjs.bak.'\$(date +%Y%m%d%H%M%S); \
  sudo cp '$NGINX_SITE_PATH' '$NGINX_SITE_PATH.bak.'\$(date +%Y%m%d%H%M%S); \
  sudo cp /tmp/campaign-lottery-gaokao-api.conf '$NGINX_SITE_PATH'; \
  sudo nginx -t; \
  sudo systemctl reload nginx; \
  pm2 reload '$REMOTE_PROJECT_DIR/ecosystem.config.cjs' --update-env; \
  pm2 save >/dev/null; \
  curl -fsS http://127.0.0.1:18100/healthz >/dev/null; \
  curl -fsS http://127.0.0.1:3000 >/dev/null"

echo "Deployment completed"