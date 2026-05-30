package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alatube/alatube/backend/internal/api"
	"github.com/alatube/alatube/backend/internal/jobs"
	"github.com/alatube/alatube/backend/internal/media"
)

func main() {
	cfg := media.RunnerConfig{
		WorkDir:    getenv("ALATUBE_WORK_DIR", "/var/lib/alatube/jobs"),
		JobTimeout: getenvDuration("ALATUBE_JOB_TIMEOUT", 5*time.Minute),
		YTDLPPath:  getenv("ALATUBE_YTDLP_PATH", "yt-dlp"),
	}
	runner := media.NewLocalRunner(cfg)

	store := jobs.NewStore(jobs.StoreConfig{
		CompletedTTL: 10 * time.Minute,
		FailedTTL:    15 * time.Minute,
		WorkDir:      cfg.WorkDir,
		QueueSize:    16,
		MaxWorkers:   1,
		Runner:       runner,
	})
	go store.CleanupLoop(1 * time.Minute)
	go store.StartWorkers()

	handler := api.NewServer(store, runner)
	addr := getenv("ALATUBE_ADDR", ":8080")
	log.Printf("alatube api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler.Routes()))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
