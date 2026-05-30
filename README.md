# AlaTube

AlaTube is a private SvelteKit PWA plus Go API for YouTube URL analysis and short-lived, single-use media jobs.

The intended deployment is deliberately simple:

- Frontend: Cloudflare Pages, connected to the GitHub repo, serving the static SvelteKit build.
- Backend: one Go binary on an Ubuntu server, managed by `systemd`.
- Go runtime: official Go 1.26.0 tarball installed under `/usr/local/go`.
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
