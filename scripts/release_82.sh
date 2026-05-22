#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST="ubuntu@82.156.54.232"
REMOTE_ROOT="/home/ubuntu/campaign-lottery-platform"
LOCAL_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SSH_BIN=(ssh)
SCP_BIN=(scp)

if [[ -n "${SSHPASS:-}" ]] && command -v sshpass >/dev/null 2>&1; then
	SSH_BIN=(sshpass -e ssh)
	SCP_BIN=(sshpass -e scp)
fi

echo "==> build backend"
cd "$LOCAL_ROOT/backend"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o campaign-lottery-api ./cmd/server/main.go

echo "==> ensure remote directories"
"${SSH_BIN[@]}" "$REMOTE_HOST" "mkdir -p $REMOTE_ROOT/backend $REMOTE_ROOT/frontend/h5 $REMOTE_ROOT/frontend/admin $REMOTE_ROOT/deploy $REMOTE_ROOT/sql"

echo "==> sync files"
"${SCP_BIN[@]}" campaign-lottery-api "$REMOTE_HOST:$REMOTE_ROOT/backend/"
"${SCP_BIN[@]}" .env.example "$REMOTE_HOST:$REMOTE_ROOT/backend/.env.example"
"${SCP_BIN[@]}" "$LOCAL_ROOT/sql/schema.mysql.sql" "$REMOTE_HOST:$REMOTE_ROOT/sql/schema.mysql.sql"
"${SCP_BIN[@]}" -r "$LOCAL_ROOT/frontend/h5/." "$REMOTE_HOST:$REMOTE_ROOT/frontend/h5/"
"${SCP_BIN[@]}" -r "$LOCAL_ROOT/frontend/admin/." "$REMOTE_HOST:$REMOTE_ROOT/frontend/admin/"
"${SCP_BIN[@]}" "$LOCAL_ROOT/deploy/systemd/campaign-lottery-platform.service" "$REMOTE_HOST:$REMOTE_ROOT/deploy/"
"${SCP_BIN[@]}" "$LOCAL_ROOT/deploy/nginx/campaign-lottery-platform.conf" "$REMOTE_HOST:$REMOTE_ROOT/deploy/"

echo "==> remote install hints"
echo "1. copy $REMOTE_ROOT/backend/.env.example to $REMOTE_ROOT/backend/.env and fill DB/Redis credentials"
echo "2. import schema: sudo mysql < $REMOTE_ROOT/sql/schema.mysql.sql"
echo "3. sudo cp $REMOTE_ROOT/deploy/campaign-lottery-platform.service /etc/systemd/system/"
echo "4. sudo cp $REMOTE_ROOT/deploy/campaign-lottery-platform.conf /etc/nginx/conf.d/"
echo "5. sudo systemctl daemon-reload && sudo systemctl enable --now campaign-lottery-platform.service"
echo "6. sudo nginx -t && sudo systemctl reload nginx"