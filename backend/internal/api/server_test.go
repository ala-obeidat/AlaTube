package api

import (
	"context"
	"net/http"
	"net/http/httptest"
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

