# AlaTube

AlaTube is a private SvelteKit PWA plus Go API for YouTube URL analysis and short-lived, single-use media jobs.

The intended deployment is deliberately simple:

- Frontend: Cloudflare Pages, connected to the GitHub repo, serving the static SvelteKit build.
- Backend: one Go binary on an Ubuntu server, managed by `systemd`.
- Go runtime: official Go 1.26.3 tarball installed under `/usr/local/go`.
- Media tools: `yt-dlp` in a Python virtual environment and `ffmpeg` from apt, run under the hardened `alatube` service account.

## Project Layout

- `frontend/`: SvelteKit with `adapter-static`, PWA manifest, service worker share target, and static build output.
- `backend/`: Go API server. It owns every `/api/*` route, validates canonical YouTube input, manages jobs, streams SSE progress, and enforces single-use download claims.
- `deploy/systemd/`: Ubuntu `systemd` unit and environment template.
- `deploy/caddy/`: Minimal Caddy reverse proxy for the API domain.
- `scripts/`: Ubuntu install/update helpers.

## Cloudflare Pages

Connect Cloudflare Pages to the GitHub repository and use:

- Root directory: repository root
- Build command: `npm install && npm --workspace frontend run build`
- Build output directory: `frontend/build`
- Node version: `24`
- Environment variable: `PUBLIC_API_BASE_URL=https://api.alatube.example.com`

The frontend is static. It does not need SSR.

## Ubuntu Backend

On the server:

```bash
sudo mkdir -p /opt/alatube
sudo git clone https://github.com/ala-obeidat/AlaTube.git /opt/alatube/src
cd /opt/alatube/src
sudo bash scripts/ubuntu-install.sh
```

Then edit:

```bash
sudo nano /etc/alatube/alatube.env
```

Set `ALATUBE_ALLOWED_ORIGINS` to your Cloudflare Pages production domain and any custom frontend domain:

```ini
ALATUBE_ALLOWED_ORIGINS=https://your-project.pages.dev,https://alatube.example.com
```

Restart:

```bash
sudo systemctl restart alatube
sudo systemctl status alatube
```

## Caddy API Proxy

Install the Caddyfile after replacing `api.alatube.example.com`:

```bash
sudo cp deploy/caddy/Caddyfile /etc/caddy/Caddyfile
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

Point DNS for the API hostname to the Ubuntu server.

## Security Notes

The non-container deployment uses `systemd` hardening instead of Docker:

- dedicated non-login `alatube` user
- `NoNewPrivileges=true`
- strict filesystem protection
- writable access only to `/var/lib/alatube` and `/var/cache/alatube`
- dropped Linux capabilities
- CPU, memory, and task limits
- private tmp directory
- strict Go job timeout
- canonical YouTube-only validation before `yt-dlp` execution

This is intentionally simpler than Docker, but it is not the same isolation boundary as a locked-down container. For stronger outbound network control, add host firewall rules for the `alatube` user or put the API behind Cloudflare Access and a server firewall that only permits required egress.

## API

- `POST /api/analyze`: `{ "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ" }`
- `POST /api/jobs`: `{ "videoId": "dQw4w9WgXcQ", "format": { "videoFormatId": "136", "audioFormatId": "140" } }`
- `GET /api/jobs/{id}/events`: `text/event-stream` job events.
- `GET /api/jobs/{id}/download`: one-time completed-file download.
- `DELETE /api/jobs/{id}`: cancel/purge queued metadata and temporary files.

Errors use:

```json
{
  "error": {
    "code": "invalid_youtube_url",
    "message": "A valid YouTube video URL is required.",
    "details": { "field": "url" },
    "requestId": "req_..."
  }
}
```

## Local Frontend

```powershell
npm install
npm --workspace frontend run dev
```

## Local Backend

This machine needs Go 1.22+:

```powershell
cd backend
go test ./...
go run ./cmd/alatube
```

Set `PUBLIC_API_BASE_URL=http://localhost:8080` for a frontend build that talks directly to a local backend.

## Optional API Token

`ALATUBE_API_TOKEN` on the backend and `PUBLIC_API_TOKEN` on the frontend gate the API behind a shared secret. The middleware bypasses when the backend env var is empty, so the gate is off by default. `/api/health` stays exempt for liveness probes. The token can travel as `Authorization: Bearer <token>` for `fetch` or as `?token=<token>` for `EventSource` and download links — the frontend uses both automatically. This is obscurity-grade defense-in-depth, not real auth; a public bundle exposes the token to anyone who reads the JS, so prefer Cloudflare Access for strong access control. Rollout sequence to avoid breaking the live site:

1. Generate a token: `openssl rand -hex 32`.
2. Add `PUBLIC_API_TOKEN=<token>` in Cloudflare Pages → Settings → Environment variables and trigger a rebuild. The frontend now sends it on every request.
3. Once the Pages rebuild is live, set `ALATUBE_API_TOKEN=<token>` in `/etc/alatube/alatube.env` on the backend and `sudo systemctl restart alatube`. From this point the API rejects requests without the token.

## YouTube Cookie Rotation

The backend's `ALATUBE_YTDLP_COOKIES` path lets `yt-dlp` authenticate to YouTube and survive bot-challenge prompts on VPS IPs. Cookies expire on YouTube's session schedule, so periodic rotation is required.

To rotate from Windows, set the SSH target and key once (env vars in your user profile, or pass `-Server`/`-Key` each time):

```powershell
[Environment]::SetEnvironmentVariable('ALATUBE_SERVER', 'root@your.server', 'User')
[Environment]::SetEnvironmentVariable('ALATUBE_SSH_KEY', 'C:\path\to\private\key', 'User')
.\scripts\rotate-cookies.ps1 -Path C:\Users\you\Downloads\cookies.txt
```

The script validates the Netscape header, uploads the new cookies to `/etc/alatube/cookies.txt.new`, runs `scripts/rotate-cookies-remote.sh` on the server to back up the old file, chmod to `0600`, atomically swap, and smoke-test `yt-dlp` against a known video before reporting success.

To install the daily health-check timer (logs an `alatube-cookie-check` line to the journal so cookie expiry surfaces before users see 502s):

```bash
sudo install -m 0755 scripts/alatube-cookie-check.sh /usr/local/bin/alatube-cookie-check.sh
sudo install -m 0644 deploy/systemd/alatube-cookie-check.service /etc/systemd/system/alatube-cookie-check.service
sudo install -m 0644 deploy/systemd/alatube-cookie-check.timer /etc/systemd/system/alatube-cookie-check.timer
sudo systemctl daemon-reload
sudo systemctl enable --now alatube-cookie-check.timer
journalctl -u alatube-cookie-check.service -n 20
```

### Scheduled rotation (Windows Task Scheduler)

Full server-side automation is not possible — rotation requires a fresh `cookies.txt` exported from a browser logged into YouTube. The split is:

- **Server**: detection via `alatube-cookie-check.timer` (above).
- **PC**: a Windows scheduled task that watches a drop folder and rotates when a fresh file appears.

The scripts read your SSH target and key from `-Server` / `-Key` parameters or the env vars `ALATUBE_SERVER` / `ALATUBE_SSH_KEY`. There are no defaults — nothing about your prod box is committed to the repo.

Set them once in your user profile (or in the current session before installing):

```powershell
[Environment]::SetEnvironmentVariable('ALATUBE_SERVER', 'root@your.server', 'User')
[Environment]::SetEnvironmentVariable('ALATUBE_SSH_KEY', 'C:\path\to\private\key', 'User')
```

Then register the daily task in an elevated PowerShell prompt. `install-scheduled-task.ps1` bakes the server + key into the task action so the scheduled run does not depend on env vars at fire time:

```powershell
cd D:\Code\AlaTube
.\scripts\install-scheduled-task.ps1 -Server root@your.server -Key C:\path\to\private\key
```

That creates the task `AlaTube Cookie Rotation` (default fire 09:30 local), and the folders `C:\AlaTube\pending` and `C:\AlaTube\archive`. Logs land at `%LOCALAPPDATA%\AlaTube\rotate.log`. Re-run the install whenever you want to change the bound server or key.

Day-to-day loop:

1. When `journalctl -t alatube-cookie-check` shows a `FAIL` line (or whenever you want to refresh proactively), export `cookies.txt` from a logged-in YouTube tab.
2. Save it as `C:\AlaTube\pending\cookies.txt`.
3. Either wait for the next daily run, or run `.\scripts\scheduled-rotate.ps1` immediately.
4. On success the file is moved into `C:\AlaTube\archive\cookies-<timestamp>.txt` so it isn't rotated again. On failure it stays put for inspection.
