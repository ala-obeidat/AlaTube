#!/bin/bash
# Daily YouTube-cookies health check for AlaTube.
# Reads ALATUBE_YTDLP_PATH and ALATUBE_YTDLP_COOKIES from the env file,
# probes a short public video with --dump-json, and logs OK/FAIL to the
# journal under tag "alatube-cookie-check" so cookie expiry surfaces
# before users hit it via /api/analyze.
set -euo pipefail

ENV_FILE="${ALATUBE_ENV_FILE:-/etc/alatube/alatube.env}"
PROBE_URL="${ALATUBE_PROBE_URL:-https://www.youtube.com/watch?v=jNQXAC9IVRw}"

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  set -a; . "$ENV_FILE"; set +a
fi

YTDLP="${ALATUBE_YTDLP_PATH:-/usr/local/bin/yt-dlp}"
COOKIES="${ALATUBE_YTDLP_COOKIES:-}"

log() { logger -t alatube-cookie-check -p "daemon.$1" "$2"; }

if [[ ! -x "$YTDLP" ]]; then
  log err "yt-dlp not found or not executable at $YTDLP"
  exit 2
fi

ARGS=(--dump-json --no-playlist --no-cache-dir --quiet)
ERR=$(mktemp)
COOKIES_TMP=""
trap 'rm -f "$ERR" "$COOKIES_TMP"' EXIT

if [[ -n "$COOKIES" && -f "$COOKIES" ]]; then
  # yt-dlp may rewrite the cookies file on close; the unit's ProtectSystem=strict
  # makes /etc read-only. Work on a writable copy in the unit's PrivateTmp.
  COOKIES_TMP=$(mktemp)
  cp "$COOKIES" "$COOKIES_TMP"
  ARGS+=(--cookies "$COOKIES_TMP")
fi

if "$YTDLP" "${ARGS[@]}" "$PROBE_URL" > /dev/null 2> "$ERR"; then
  log info "OK probe=$PROBE_URL cookies=${COOKIES:-none}"
  exit 0
fi

REASON=$(tail -c 400 "$ERR" | tr '\n' ' ')
log err "FAIL probe=$PROBE_URL cookies=${COOKIES:-none} stderr_tail=\"$REASON\""
exit 1
