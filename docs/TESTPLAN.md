# AlaTube Test Plan

Date: 2026-05-30

## Coverage Matrix

| Area | Scenario | Source | Status | How Tested |
|---|---|---:|---:|---|
| Baseline | API health | Live | Pass | `GET https://alatube-api.alaobeidat.com/api/health` returned 200 in ~0.21-0.25s. |
| Baseline | API response headers | Live | Pass | Captured in `docs/headers-health.txt`; response is proxied by Caddy/Cloudflare and echoes `X-Request-Id`. |
| CORS | Allowed frontend origin preflight | Live | Pass | `OPTIONS /api/analyze` from `https://alatube.alaobeidat.com` returned ACAO and credentials headers. |
| CORS | Foreign origin preflight | Live | Pass | `OPTIONS /api/analyze` from `https://evil.example` returned no ACAO/credentials headers. |
| CORS | No configured allowlist must not echo arbitrary origin | Local | Pass | `TestCORSRequiresExplicitAllowedOrigin`. |
| CORS | Configured allowlist permits frontend origin | Local | Pass | `TestCORSAllowsConfiguredOrigin`. |
| `POST /api/analyze` | Playlist-only URL rejected | Live + Local | Pass | Live returned 400 `invalid_youtube_url`; local URL parser test covers playlist rejection. |
| `POST /api/analyze` | Wrong content type rejected | Live | Pass | Live returned 415 `unsupported_media_type`. |
| `POST /api/analyze` | Malformed JSON rejected | Live | Pass | Live returned 400 `invalid_json`. |
| `POST /api/analyze` | Unknown JSON field rejected | Live | Pass | Live returned 400 `invalid_json` through `DisallowUnknownFields`. |
| `POST /api/analyze` | Happy path test video | Live | Pass | Live analyzed `https://youtu.be/paLT7JuHPBc` in ~5.25s. |
| URL validation | Userinfo host bypass rejected | Live + Local | Pass | Live returned 400; local parser rejects non-allowlisted host. |
| URL validation | Explicit port rejected | Local | Pass | Added `explicit port rejected` test. Live before fix accepted this and canonicalized it. |
| URL validation | Trailing-dot hostname rejected | Local | Pass | Added `trailing dot host rejected` test. |
| URL validation | Uppercase hostname accepted | Local | Pass | Added `uppercase host accepted` test. |
| `POST /api/jobs` | Option-looking `videoFormatId` rejected | Local | Pass | `TestCreateJobRejectsOptionLikeFormatID`. Live before fix accepted and queued these jobs. |
| `POST /api/jobs` | Argument-injection-looking payloads | Live | Observed | `--exec=id`, `--config-location=/etc/alatube/cookies.txt`, and `--load-info-json=/etc/passwd` were accepted before local fix; each job was deleted and health stayed 200. |
| `GET /api/jobs/{id}/download` | Single-use claim | Local | Pass | Existing `TestDownloadClaimIsSingleUse`. |
| `GET /api/jobs/{id}/download` | 100-way claim race | Local | Pass | Added `TestDownloadClaimRaceHasSingleWinner`; exactly one claim wins. |
| Cleanup | Expired completed job file removal | Local | Pass | Added `TestCleanupRemovesExpiredCompletedJobFile`. |
| Frontend | Type/check/build | Local | Pass | `npm --workspace frontend run check`, `test`, and `build`. |
| Go tests | Unit tests | Local | Pass | `go test ./...` with Go 1.22.12 and Go 1.25.10. |
| Go race | Race detector | Local | Blocked | `go test -race ./...` requires CGO and this Windows host has no `gcc`. |
| Go vet | Vet | Local | Pass | `go vet ./...`. |
| staticcheck | Static analysis | Local | Pass | `staticcheck ./...` with staticcheck 2026.1. |
| gosec | Security static analysis | Local | Pass | `gosec ./...` found 0 issues after fixes; report in `docs/gosec-report-go1.25.10.json`. |
| govulncheck | Vulnerability scan | Local | Pass | Go 1.25.10 scan found no vulnerabilities; report in `docs/govulncheck-go1.25.10.txt`. |

## Commands Run

```powershell
npm --workspace frontend run check
npm --workspace frontend run test
npm --workspace frontend run build

$env:GOROOT='D:\Code\AlaTube\.tools\go1.25.10'
$env:PATH="$env:GOROOT\bin;D:\Code\AlaTube\.tools\gobin;$env:PATH"
go test ./...
go vet ./...
staticcheck ./...
gosec -fmt=json -out ..\docs\gosec-report-go1.25.10.json ./...
govulncheck ./...
```

## Notes

- Live request volume stayed below the requested thresholds.
- No live stop condition was triggered.
- Every live job created during argument-injection probing was deleted.
- The first two live log entries are local harness failures before curl sent a usable URL; they are kept in `docs/LIVE-RUN-LOG.md` for transparency.
