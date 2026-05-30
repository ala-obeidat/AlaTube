#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-/opt/alatube/src}"
APP_DIR="${APP_DIR:-/opt/alatube}"

export PATH="/usr/local/go/bin:${PATH}"

cd "${REPO_DIR}"
git pull --ff-only

cd "${REPO_DIR}/backend"
go test ./...
go build -trimpath -ldflags="-s -w" -o "${APP_DIR}/bin/alatube" ./cmd/alatube

"${APP_DIR}/venv/bin/pip" install --upgrade yt-dlp

systemctl restart alatube
systemctl status --no-pager alatube
