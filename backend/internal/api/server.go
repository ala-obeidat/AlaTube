package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/alatube/alatube/backend/internal/jobs"
	"github.com/alatube/alatube/backend/internal/media"
	"github.com/alatube/alatube/backend/internal/security"
)

type Server struct {
	store          *jobs.Store
	runner         media.Runner
	allowedOrigins map[string]struct{}
	apiToken       string
}

func NewServer(store *jobs.Store, runner media.Runner) *Server {
	return &Server{
		store:          store,
		runner:         runner,
		allowedOrigins: parseAllowedOrigins(os.Getenv("ALATUBE_ALLOWED_ORIGINS")),
		apiToken:       strings.TrimSpace(os.Getenv("ALATUBE_API_TOKEN")),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/analyze", s.analyze)
	mux.HandleFunc("POST /api/jobs", s.createJob)
	mux.HandleFunc("GET /api/jobs/{id}/events", s.jobEvents)
	mux.HandleFunc("GET /api/jobs/{id}/download", s.download)
	mux.HandleFunc("DELETE /api/jobs/{id}", s.deleteJob)
	return requestID(s.cors(s.requireAPIToken(mux)))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) analyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	canon, err := security.ParseYouTubeURL(req.URL)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_youtube_url", "A valid YouTube video URL is required.", map[string]string{"field": "url"})
		return
	}
	result, err := s.runner.Analyze(r.Context(), canon.URL, canon.ID)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "media_analysis_failed", "Media analysis failed.", nil)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VideoID string `json:"videoId"`
		Format  struct {
			VideoFormatID string `json:"videoFormatId"`
			AudioFormatID string `json:"audioFormatId"`
		} `json:"format"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	canon, err := security.CanonicalURLFromID(req.VideoID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_youtube_video_id", "A valid YouTube video ID is required.", map[string]string{"field": "videoId"})
		return
	}
	if req.Format.VideoFormatID == "" {
		writeError(w, r, http.StatusBadRequest, "missing_format", "A video format is required.", map[string]string{"field": "format.videoFormatId"})
		return
	}
	if !validFormatID(req.Format.VideoFormatID) {
		writeError(w, r, http.StatusBadRequest, "invalid_format", "The video format is invalid.", map[string]string{"field": "format.videoFormatId"})
		return
	}
	if req.Format.AudioFormatID != "" && !validFormatID(req.Format.AudioFormatID) {
		writeError(w, r, http.StatusBadRequest, "invalid_format", "The audio format is invalid.", map[string]string{"field": "format.audioFormatId"})
		return
	}
	job, err := s.store.Create(canon.ID, canon.URL, req.Format.VideoFormatID, req.Format.AudioFormatID)
	if err != nil {
		writeError(w, r, http.StatusTooManyRequests, "queue_full", "The job queue is full. Try again later.", nil)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"jobId":       job.ID,
		"state":       job.State,
		"eventsUrl":   "/api/jobs/" + job.ID + "/events",
		"downloadUrl": nil,
		"expiresAt":   nil,
	})
}

func (s *Server) jobEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, unsubscribe, history, ok := s.store.Subscribe(id)
	if !ok {
		writeError(w, r, http.StatusNotFound, "job_not_found", "Job not found.", nil)
		return
	}
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, event := range history {
		writeSSE(w, event)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			writeSSE(w, event)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path, err := s.store.ClaimDownload(id)
	if err != nil {
		switch err.Error() {
		case "not_found":
			writeError(w, r, http.StatusNotFound, "job_not_found", "Job not found.", nil)
		case "already_claimed":
			writeError(w, r, http.StatusConflict, "download_already_claimed", "This download has already been claimed.", nil)
		default:
			writeError(w, r, http.StatusConflict, "download_not_ready", "The job is not ready for download.", nil)
		}
		return
	}
	http.ServeFile(w, r, path)
	if r.Context().Err() == nil {
		s.store.MarkDownloadFinished(id)
	}
}

func (s *Server) deleteJob(w http.ResponseWriter, r *http.Request) {
	if !s.store.Delete(r.PathValue("id")) {
		writeError(w, r, http.StatusNotFound, "job_not_found", "Job not found.", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json.", nil)
		return false
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.", nil)
		return false
	}
	return true
}

func writeSSE(w http.ResponseWriter, payload interface{}) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: job\n")
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details interface{}) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":      code,
			"message":   message,
			"details":   details,
			"requestId": requestIDFrom(r.Context()),
		},
	})
}

type requestIDKey struct{}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = time.Now().UTC().Format("20060102150405.000000000")
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFrom(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := s.allowedOrigin(r); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-Id, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowedOrigin(r *http.Request) string {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return ""
	}
	if _, ok := s.allowedOrigins[origin]; ok {
		return origin
	}
	return ""
}

func parseAllowedOrigins(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(part)
		if origin != "" {
			out[origin] = struct{}{}
		}
	}
	return out
}

var formatIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:+-]{1,64}$`)

func validFormatID(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && formatIDPattern.MatchString(value)
}

func (s *Server) requireAPIToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.apiToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		provided := ""
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			provided = h[len("Bearer "):]
		} else if t := r.URL.Query().Get("token"); t != "" {
			provided = t
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.apiToken)) != 1 {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "API token required.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
