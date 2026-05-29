#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
  exec bash "$0" "$@"
fi

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_HOST="${DEPLOY_HOST:-82.156.54.232}"
REMOTE_USER="${DEPLOY_USER:-ubuntu}"
REMOTE_PROJECT_DIR="${REMOTE_PROJECT_DIR:-/home/ubuntu/campaign-lottery-next}"
NGINX_SITE_PATH="${NGINX_SITE_PATH:-/etc/nginx/sites-available/gaokao-api}"
SSH_PASSWORD="${DEPLOY_PASSWORD:-Zhao42818664051}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-zhao42818664051}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-CampaignDB_20260521!}"
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

python3 - "$tmp_dir/ecosystem.config.cjs" "$tmp_dir/gaokao-api.conf" "$ADMIN_PASSWORD" "$MYSQL_PASSWORD" "$CORS_ALLOW_ORIGIN" "$REMOTE_PROJECT_DIR" <<'PY'
from pathlib import Path
import sys

ecosystem_path = Path(sys.argv[1])
nginx_path = Path(sys.argv[2])
admin_password = sys.argv[3]
mysql_password = sys.argv[4]
cors_allow_origin = sys.argv[5]
remote_project_dir = sys.argv[6]
template_dir = "/home/ubuntu/campaign-lottery-next"

def patch_deploy_file(path: Path) -> None:
    text = path.read_text()
    if remote_project_dir != template_dir:
        text = text.replace(template_dir, remote_project_dir)
    path.write_text(text)

patch_deploy_file(ecosystem_path)
patch_deploy_file(nginx_path)

text = ecosystem_path.read_text()
text = text.replace("ADMIN_PASSWORD: 'change-me'", f"ADMIN_PASSWORD: '{admin_password}'")
text = text.replace("MYSQL_PASSWORD: 'change-me'", f"MYSQL_PASSWORD: '{mysql_password}'")
text = text.replace("CORS_ALLOW_ORIGIN: '*'", f"CORS_ALLOW_ORIGIN: '{cors_allow_origin}'")
ecosystem_path.write_text(text)
PY

echo "[1/6] Sync source code to $REMOTE_USER@$REMOTE_HOST"
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

echo "[2/6] Upload deploy templates"
"${scp_cmd[@]}" "$tmp_dir/ecosystem.config.cjs" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PROJECT_DIR/ecosystem.config.cjs"
"${scp_cmd[@]}" "$tmp_dir/gaokao-api.conf" "$REMOTE_USER@$REMOTE_HOST:/tmp/campaign-lottery-gaokao-api.conf"

echo "[3/6] Install backend dependencies and run migrations"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/backend-server' && npm install --no-fund --no-audit && npm run migrate && \
  if [ ! -f config/payment.config.json ]; then cp config/payment.config.mock.json config/payment.config.json; fi"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/payment-module' && rm -rf node_modules dist && ln -s ../backend-server/node_modules node_modules && ../backend-server/node_modules/.bin/tsc -p tsconfig.build.json"
"${ssh_cmd[@]}" "rm -rf '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery/payment-module' && mkdir -p '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery' && cp -R '$REMOTE_PROJECT_DIR/payment-module' '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery/payment-module' && rm -rf '$REMOTE_PROJECT_DIR/backend-server/node_modules/@campaign-lottery/payment-module/node_modules'"

echo "[4/6] Verify synced front-page source (no legacy guest nickname login UI)"
"${ssh_cmd[@]}" "if grep -q '输入昵称' '$REMOTE_PROJECT_DIR/front-page/src/features/lottery/lottery-app.tsx'; then echo 'ERROR: remote lottery-app.tsx still has legacy login UI — check REMOTE_PROJECT_DIR and local branch' >&2; exit 1; fi"

echo "[5/6] Build frontend and backend (clean .next to avoid stale UI bundles)"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/backend-server' && rm -rf .next && npm run build"
"${ssh_cmd[@]}" "cd '$REMOTE_PROJECT_DIR/front-page' && rm -rf .next && npm install --no-fund --no-audit && npm run build"
"${ssh_cmd[@]}" "if grep -rq '输入昵称' '$REMOTE_PROJECT_DIR/front-page/.next' 2>/dev/null; then echo 'ERROR: front-page build still contains legacy login UI (输入昵称)' >&2; exit 1; fi"

echo "[6/6] Replace PM2/nginx config and restart services"
"${ssh_cmd[@]}" "set -euo pipefail; \
  cp '$REMOTE_PROJECT_DIR/ecosystem.config.cjs' '$REMOTE_PROJECT_DIR/ecosystem.config.cjs.bak.'\$(date +%Y%m%d%H%M%S); \
  sudo cp '$NGINX_SITE_PATH' '$NGINX_SITE_PATH.bak.'\$(date +%Y%m%d%H%M%S); \
  sudo cp /tmp/campaign-lottery-gaokao-api.conf '$NGINX_SITE_PATH'; \
  sudo nginx -t; \
  sudo systemctl reload nginx; \
  pm2 delete campaign-lottery-api campaign-lottery-front >/dev/null 2>&1 || true; \
  pm2 start '$REMOTE_PROJECT_DIR/ecosystem.config.cjs' --update-env; \
  pm2 save >/dev/null; \
  api_ready=0; \
  for i in \$(seq 1 30); do \
    if curl -sS --connect-timeout 2 --max-time 5 http://127.0.0.1:18100/healthz 2>/dev/null | grep -q 'campaign-lottery-backend-server'; then \
      api_ready=1; \
      break; \
    fi; \
    sleep 2; \
  done; \
  if [ \"\$api_ready\" -ne 1 ]; then \
    echo 'ERROR: API not listening on 127.0.0.1:18100 after PM2 restart (check pm2 logs campaign-lottery-api)' >&2; \
    pm2 logs campaign-lottery-api --lines 40 --nostream >&2 || true; \
    exit 1; \
  fi; \
  front_ready=0; \
  for i in \$(seq 1 30); do \
    if curl -fsS --connect-timeout 2 --max-time 5 http://127.0.0.1:3000/ >/dev/null 2>&1; then \
      front_ready=1; \
      break; \
    fi; \
    sleep 2; \
  done; \
  if [ \"\$front_ready\" -ne 1 ]; then \
    echo 'ERROR: front-page not listening on 127.0.0.1:3000 after PM2 restart (check pm2 logs campaign-lottery-front)' >&2; \
    pm2 logs campaign-lottery-front --lines 40 --nostream >&2 || true; \
    exit 1; \
  fi; \
  if curl -fsS http://127.0.0.1:3000/ | grep -q '输入昵称'; then echo 'ERROR: running front still serves legacy login HTML' >&2; exit 1; fi"

echo "Deployment completed to $REMOTE_PROJECT_DIR"