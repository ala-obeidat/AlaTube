#!/bin/bash
# Run on the prod box. Atomically swap in a fresh cookies file uploaded to
# /etc/alatube/cookies.txt.new, back up the old one, and smoke-test yt-dlp.
#
# Usage: sudo bash rotate-cookies-remote.sh [SMOKE_URL]
set -euo pipefail

SMOKE_URL="${1:-https://www.youtube.com/watch?v=jNQXAC9IVRw}"
COOKIES=/etc/alatube/cookies.txt
NEW=/etc/alatube/cookies.txt.new
YTDLP="${ALATUBE_YTDLP_PATH:-/usr/local/bin/yt-dlp}"

if [[ ! -f "$NEW" ]]; then
  echo "missing upload: $NEW" >&2
  exit 1
fi

FIRST=$(head -c 200 "$NEW")
case "$FIRST" in
  *"Netscape HTTP Cookie File"*) ;;
  *) echo "not a Netscape cookies file" >&2; exit 1 ;;
esac

chown root:root "$NEW"
chmod 600 "$NEW"

if [[ -f "$COOKIES" ]]; then
  STAMP=$(date -u +%Y%m%dT%H%M%SZ)
  cp -a "$COOKIES" "${COOKIES}.bak.${STAMP}"
fi

mv "$NEW" "$COOKIES"
ls -la "$COOKIES"

OUT=$(mktemp)
ERR=$(mktemp)
trap 'rm -f "$OUT" "$ERR"' EXIT

if "$YTDLP" --dump-json --no-playlist --no-cache-dir --cookies "$COOKIES" "$SMOKE_URL" > "$OUT" 2> "$ERR"; then
  BYTES=$(wc -c < "$OUT")
  echo "smoke OK: yt-dlp returned ${BYTES} bytes for ${SMOKE_URL}"
else
  echo "smoke FAIL: yt-dlp exited non-zero" >&2
  tail -c 800 "$ERR" >&2
  exit 1
fi
