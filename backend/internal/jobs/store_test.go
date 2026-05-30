package jobs

import (
	"context"
	"os"
	"path/filepath"
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

