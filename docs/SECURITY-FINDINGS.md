# AlaTube Security Findings

Date: 2026-05-30

## High

### `yt-dlp` Format IDs Accepted Option-Looking Values

- Severity: High
- Source: Live + local
- Where: `backend/internal/api/server.go`, `backend/internal/media/runner.go`
- Evidence: Live `POST /api/jobs` accepted `videoFormatId` values `--exec=id`, `--config-location=/etc/alatube/cookies.txt`, and `--load-info-json=/etc/passwd`, returning 202 for each. Jobs were immediately deleted and `/api/health` stayed 200.
- Risk: Even without shell execution, passing user-controlled strings into argv positions consumed by `yt-dlp` can convert data into options.
- Fix applied locally: `validFormatID` rejects empty values, leading `-`, and characters outside a tight format-ID character set. Regression: `TestCreateJobRejectsOptionLikeFormatID`.

### Build/Deploy Path Used Vulnerable Go 1.22 Toolchain

- Severity: High
- Source: Local static scan
- Where: `scripts/ubuntu-install.sh`, `README.md`
- Evidence: `govulncheck` under Go 1.22.12 reported 19 reachable standard-library vulnerabilities, including `net/http`, `net/url`, `crypto/tls`, and `os/exec` paths. The same scan under Go 1.25.10 reported: `No vulnerabilities found.`
- Risk: The Ubuntu apt `golang-go` package can lag behind fixed standard-library releases.
- Fix applied locally: `scripts/ubuntu-install.sh` now installs official Go 1.26.3 under `/usr/local/go`, and `scripts/ubuntu-update.sh` uses it.
- Follow-up (2026-05-30): a fresh `govulncheck` run on prod under Go 1.26.0 (apt-installed) surfaced 9 newly-reachable stdlib CVEs after the vuln DB updated on 2026-05-29 (TLS 1.3 KeyUpdate DoS, x509 chain validation, IPv6 host parsing in `net/url`, `os.File` `Root` escape in our `http.ServeFile` path). Bumped prod to Go 1.26.3 tarball under `/usr/local/go`, rebuilt the binary, and re-ran the scan: `No vulnerabilities found.` See `docs/govulncheck-go1.26.3.txt`.

## Medium

### HTTP Server Lacked Explicit Timeouts

- Severity: Medium
- Source: Local `gosec`
- Where: `backend/cmd/alatube/main.go`
- Evidence: `gosec` rule G114 flagged `http.ListenAndServe`.
- Risk: Slowloris-style clients can hold server resources longer than intended.
- Fix applied locally: replaced `http.ListenAndServe` with `http.Server` using `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`. `gosec` is now clean.

### Empty CORS Allowlist Echoed Arbitrary Origins

- Severity: Medium
- Source: Local code review + regression test
- Where: `backend/internal/api/server.go`
- Evidence: `allowedOrigin` previously returned the request origin when `ALATUBE_ALLOWED_ORIGINS` was empty.
- Risk: If the production env var is missing, the API emits credentialed CORS for arbitrary origins.
- Live status: Production appears correctly configured; `https://evil.example` received no ACAO/credentials headers.
- Fix applied locally: empty allowlist now denies all cross-origin CORS. Regressions: `TestCORSRequiresExplicitAllowedOrigin`, `TestCORSAllowsConfiguredOrigin`.

## Low

### YouTube Host Allowlist Accepted Explicit Ports

- Severity: Low
- Source: Live + local
- Where: `backend/internal/security/youtube.go`
- Evidence: Live `POST /api/analyze` accepted `https://www.youtube.com:8080/watch?v=dQw4w9WgXcQ` and canonicalized it to the normal YouTube watch URL.
- Risk: Current behavior is not SSRF-practical because the backend rebuilds the canonical URL before subprocess execution, but accepting noncanonical authorities weakens the allowlist contract.
- Fix applied locally: reject URLs with explicit ports. Regression: `explicit port rejected`.

### Trailing-Dot YouTube Hostnames Were Normalized

- Severity: Low
- Source: Local code review
- Where: `backend/internal/security/youtube.go`
- Evidence: The parser used `strings.TrimSuffix(host, ".")`.
- Risk: Trailing-dot normalization broadens accepted authority forms unnecessarily.
- Fix applied locally: reject trailing-dot hostnames. Regression: `trailing dot host rejected`.

## Info

### `gosec` Subprocess Warnings Require Context

- Severity: Info
- Source: Local `gosec`
- Where: `backend/internal/media/runner.go`
- Evidence: G204 flagged `exec.CommandContext` calls.
- Assessment: Subprocess execution is intended. The important controls are canonical URL reconstruction, format-ID validation, no shell invocation, systemd hardening, and job timeout.
- Fix applied locally: added narrow `#nosec G204` comments with rationale after adding validation.

### Race Detector Blocked on Windows Host

- Severity: Info
- Source: Local tooling
- Where: test environment
- Evidence: `go test -race ./...` failed with `cgo: C compiler "gcc" not found`.
- Recommendation: run `go test -race ./...` on the Ubuntu server or a CI Linux runner with CGO toolchain installed.

### API Authentication Is External

- Severity: Info
- Source: Architecture review
- Where: deployment boundary
- Evidence: The Go API does not implement app-layer authentication.
- Recommendation: keep the API behind Cloudflare Access, or add a lightweight `ALATUBE_API_TOKEN` middleware if Cloudflare Access is not guaranteed.
