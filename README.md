# AlaTube

AlaTube is a private static SvelteKit PWA plus Go API for YouTube URL analysis and short-lived, single-use media jobs.

## Architecture

- `frontend/`: SvelteKit with `adapter-static`, PWA manifest, service worker share target, and static build output.
- `backend/`: Go API server. It owns every `/api/*` route, validates canonical YouTube input, manages jobs, streams SSE progress, and enforces single-use download claims.
- `deploy/`: Caddy, Docker Compose, and a locked-down media runner image for yt-dlp/FFmpeg subprocess execution.

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

## Backend

This machine needs Go 1.22+ to run the backend directly:

```powershell
cd backend
go test ./...
go run ./cmd/alatube
```

## Deployment

Build the media runner first, then start the stack:

```bash
sudo mkdir -p /srv/alatube/jobs
cd deploy
docker compose --profile build-only build media-runner
docker compose up --build
```

Caddy serves the static frontend and proxies `/api/*` to the Go backend.

The media runner container is invoked with a non-root user, read-only root filesystem, `no-new-privileges`, dropped capabilities, CPU/memory/pid limits, a tmpfs temporary directory, and a strict Go context timeout. Keep the input validator as the first gate before any subprocess execution.

The backend uses the Docker socket to launch that locked-down runner. On a production VPS, prefer a Docker socket proxy or a dedicated runner daemon with only the minimal container-run capability exposed to the API process.
