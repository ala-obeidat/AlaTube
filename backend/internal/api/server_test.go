package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alatube/alatube/backend/internal/jobs"
	"github.com/alatube/alatube/backend/internal/media"
)

type runner struct{}

func (runner) Analyze(context.Context, string, string) (media.Analysis, error) {
	return media.Analysis{VideoID: "dQw4w9WgXcQ", CanonicalURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}, nil
}
func (runner) Mux(context.Context, media.JobRequest) (string, error) { return "", nil }

func TestAnalyzeRejectsPlaylistOnlyURL(t *testing.T) {
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(`{"url":"https://www.youtube.com/playlist?list=abc"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "invalid_youtube_url") {
		t.Fatalf("missing error code: %s", rec.Body.String())
	}
}

func TestCreateJobRejectsOptionLikeFormatID(t *testing.T) {
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", strings.NewReader(`{"videoId":"dQw4w9WgXcQ","format":{"videoFormatId":"--exec=id"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "invalid_format") {
		t.Fatalf("missing invalid_format error: %s", rec.Body.String())
	}
}

func TestCORSRequiresExplicitAllowedOrigin(t *testing.T) {
	t.Setenv("ALATUBE_ALLOWED_ORIGINS", "")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodOptions, "/api/analyze", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("ALATUBE_ALLOWED_ORIGINS", "https://alatube.alaobeidat.com")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodOptions, "/api/analyze", nil)
	req.Header.Set("Origin", "https://alatube.alaobeidat.com")
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://alatube.alaobeidat.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestAPITokenBypassedWhenEnvUnset(t *testing.T) {
	t.Setenv("ALATUBE_API_TOKEN", "")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health = %d, want 200", rec.Code)
	}
}

func TestAPITokenRejectsMissing(t *testing.T) {
	t.Setenv("ALATUBE_API_TOKEN", "s3cret")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(`{"url":"https://youtu.be/dQw4w9WgXcQ"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unauthorized") {
		t.Fatalf("missing unauthorized: %s", rec.Body.String())
	}
}

func TestAPITokenAcceptsBearer(t *testing.T) {
	t.Setenv("ALATUBE_API_TOKEN", "s3cret")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(`{"url":"https://youtu.be/dQw4w9WgXcQ"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer s3cret")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("bearer auth rejected: %s", rec.Body.String())
	}
}

func TestAPITokenAcceptsQueryParam(t *testing.T) {
	t.Setenv("ALATUBE_API_TOKEN", "s3cret")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodGet, "/api/jobs/abc/events?token=s3cret", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("query token rejected: %s", rec.Body.String())
	}
}

func TestAPITokenHealthExempt(t *testing.T) {
	t.Setenv("ALATUBE_API_TOKEN", "s3cret")
	store := jobs.NewStore(jobs.StoreConfig{Runner: runner{}})
	server := NewServer(store, runner{})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health = %d, want 200 (must be exempt)", rec.Code)
	}
}

func TestMain(m *testing.M) {
	os.Unsetenv("ALATUBE_ALLOWED_ORIGINS")
	os.Unsetenv("ALATUBE_API_TOKEN")
	os.Exit(m.Run())
}
