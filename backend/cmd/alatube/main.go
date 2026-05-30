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
		Image:       getenv("ALATUBE_MEDIA_IMAGE", "alatube-media-runner:local"),
		WorkDir:     getenv("ALATUBE_WORK_DIR", "/tmp/alatube-jobs"),
		JobTimeout:  getenvDuration("ALATUBE_JOB_TIMEOUT", 5*time.Minute),
		MemoryLimit: getenv("ALATUBE_MEDIA_MEMORY", "512m"),
		CPUs:        getenv("ALATUBE_MEDIA_CPUS", "1.0"),
	}

	store := jobs.NewStore(jobs.StoreConfig{
		CompletedTTL: 10 * time.Minute,
		FailedTTL:    15 * time.Minute,
		WorkDir:      cfg.WorkDir,
		QueueSize:    16,
		MaxWorkers:   1,
		Runner:       media.NewDockerRunner(cfg),
	})
	go store.CleanupLoop(1 * time.Minute)
	go store.StartWorkers()

	handler := api.NewServer(store, media.NewDockerRunner(cfg))
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
