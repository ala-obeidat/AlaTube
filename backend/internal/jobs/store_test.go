package jobs

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alatube/alatube/backend/internal/media"
)

type fakeRunner struct{}

func (fakeRunner) Analyze(context.Context, string, string) (media.Analysis, error) {
	return media.Analysis{}, nil
}

func (fakeRunner) Mux(_ context.Context, req media.JobRequest) (string, error) {
	_ = os.MkdirAll(req.OutputDirectory, 0o700)
	path := filepath.Join(req.OutputDirectory, "output.mp4")
	return path, os.WriteFile(path, []byte("ok"), 0o600)
}

func TestDownloadClaimIsSingleUse(t *testing.T) {
	store := NewStore(StoreConfig{CompletedTTL: time.Minute, FailedTTL: time.Minute, QueueSize: 1, MaxWorkers: 1, Runner: fakeRunner{}})
	job, err := store.Create("dQw4w9WgXcQ", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "136", "140")
	if err != nil {
		t.Fatal(err)
	}
	store.runJob(job.ID)

	if _, err := store.ClaimDownload(job.ID); err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if _, err := store.ClaimDownload(job.ID); err == nil {
		t.Fatal("second claim unexpectedly succeeded")
	}
}

func TestDownloadClaimRaceHasSingleWinner(t *testing.T) {
	store := NewStore(StoreConfig{CompletedTTL: time.Minute, FailedTTL: time.Minute, QueueSize: 1, MaxWorkers: 1, Runner: fakeRunner{}})
	job, err := store.Create("dQw4w9WgXcQ", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "136", "140")
	if err != nil {
		t.Fatal(err)
	}
	store.runJob(job.ID)

	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.ClaimDownload(job.ID); err == nil {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if winners != 1 {
		t.Fatalf("winners = %d, want 1", winners)
	}
}

func TestCleanupRemovesExpiredCompletedJobFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.mp4")
	if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewStore(StoreConfig{CompletedTTL: time.Minute, FailedTTL: time.Minute, QueueSize: 1, MaxWorkers: 1, Runner: fakeRunner{}})
	expiresAt := time.Now().UTC().Add(-time.Second)
	store.mu.Lock()
	store.jobs["expired"] = &Job{
		ID:          "expired",
		State:       StateCompleted,
		FilePath:    path,
		ExpiresAt:   &expiresAt,
		subscribers: make(map[chan Event]struct{}),
	}
	store.mu.Unlock()

	store.Cleanup(time.Now().UTC())
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected cleanup to remove file, stat err = %v", err)
	}
}
