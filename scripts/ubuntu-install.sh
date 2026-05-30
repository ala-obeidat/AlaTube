#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-/opt/alatube/src}"
APP_DIR="${APP_DIR:-/opt/alatube}"
SERVICE_USER="${SERVICE_USER:-alatube}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run this script with sudo." >&2
  exit 1
fi

apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates \
  caddy \
  ffmpeg \
  git \
  golang-go \
  python3-venv

if ! id "${SERVICE_USER}" >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/alatube --create-home --shell /usr/sbin/nologin "${SERVICE_USER}"
fi

install -d -o root -g root -m 0755 "${APP_DIR}" "${APP_DIR}/bin" /etc/alatube
install -d -o "${SERVICE_USER}" -g "${SERVICE_USER}" -m 0750 /var/lib/alatube/jobs /var/cache/alatube

if [[ ! -d "${REPO_DIR}/.git" ]]; then
  echo "Clone your GitHub repo to ${REPO_DIR} first, or set REPO_DIR to an existing checkout." >&2
  exit 1
fi

cd "${REPO_DIR}/backend"
go test ./...
go build -trimpath -ldflags="-s -w" -o "${APP_DIR}/bin/alatube" ./cmd/alatube
chown root:root "${APP_DIR}/bin/alatube"
chmod 0755 "${APP_DIR}/bin/alatube"

python3 -m venv "${APP_DIR}/venv"
"${APP_DIR}/venv/bin/pip" install --upgrade pip yt-dlp

if [[ ! -f /etc/alatube/alatube.env ]]; then
  install -m 0640 "${REPO_DIR}/deploy/systemd/alatube.env.example" /etc/alatube/alatube.env
  chown root:"${SERVICE_USER}" /etc/alatube/alatube.env
  echo "Edit /etc/alatube/alatube.env before exposing the service publicly."
fi

install -m 0644 "${REPO_DIR}/deploy/systemd/alatube.service" /etc/systemd/system/alatube.service
systemctl daemon-reload
systemctl enable --now alatube

echo "AlaTube backend installed. Check status with: systemctl status alatube"

