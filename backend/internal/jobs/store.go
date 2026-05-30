package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alatube/alatube/backend/internal/media"
)

type State string

const (
	StateQueued     State = "queued"
	StateProcessing State = "processing"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"
	StateExpired    State = "expired"
)

type Event struct {
	JobID       string      `json:"jobId"`
	State       State       `json:"state"`
	Progress    float64     `json:"progress"`
	Message     string      `json:"message"`
	DownloadURL string      `json:"downloadUrl,omitempty"`
	ExpiresAt   *time.Time  `json:"expiresAt,omitempty"`
	Error       interface{} `json:"error"`
}

type Job struct {
	ID           string
	VideoID      string
	CanonicalURL string
	VideoFormat  string
	AudioFormat  string
	State        State
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    *time.Time
	FilePath     string
	Claimed      bool
	Err          string
	events       []Event
	subscribers  map[chan Event]struct{}
}

type StoreConfig struct {
	CompletedTTL time.Duration
	FailedTTL    time.Duration
	WorkDir      string
	QueueSize    int
	MaxWorkers   int
	Runner       media.Runner
}

type Store struct {
	cfg   StoreConfig
	mu    sync.Mutex
	jobs  map[string]*Job
	queue chan string
}

func NewStore(cfg StoreConfig) *Store {
	if cfg.QueueSize == 0 {
		cfg.QueueSize = 16
	}
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = 1
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = filepath.Join(os.TempDir(), "alatube-jobs")
	}
	return &Store{
		cfg:   cfg,
		jobs:  make(map[string]*Job),
		queue: make(chan string, cfg.QueueSize),
	}
}

func (s *Store) Create(videoID, canonicalURL, videoFormat, audioFormat string) (*Job, error) {
	job := &Job{
		ID:           newID(),
		VideoID:      videoID,
		CanonicalURL: canonicalURL,
		VideoFormat:  videoFormat,
		AudioFormat:  audioFormat,
		State:        StateQueued,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		subscribers:  make(map[chan Event]struct{}),
	}

	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	s.publish(job, 0, "Queued", nil)

	select {
	case s.queue <- job.ID:
		return job, nil
	default:
		s.mu.Lock()
		delete(s.jobs, job.ID)
		s.mu.Unlock()
		return nil, errors.New("queue is full")
	}
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	copy := *job
	return &copy, true
}

func (s *Store) Subscribe(id string) (<-chan Event, func(), []Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, nil, nil, false
	}
	ch := make(chan Event, 8)
	job.subscribers[ch] = struct{}{}
	history := append([]Event(nil), job.events...)
	return ch, func() {
		s.mu.Lock()
		delete(job.subscribers, ch)
		s.mu.Unlock()
	}, history, true
}

func (s *Store) ClaimDownload(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return "", errors.New("not_found")
	}
	if job.State != StateCompleted || job.FilePath == "" {
		return "", errors.New("not_ready")
	}
	if job.Claimed {
		return "", errors.New("already_claimed")
	}
	job.Claimed = true
	job.UpdatedAt = time.Now().UTC()
	return job.FilePath, nil
}

func (s *Store) MarkDownloadFinished(id string) {
	s.mu.Lock()
	job, ok := s.jobs[id]
	if ok {
		path := job.FilePath
		job.FilePath = ""
		job.State = StateExpired
		job.UpdatedAt = time.Now().UTC()
		s.mu.Unlock()
		_ = os.Remove(path)
		return
	}
	s.mu.Unlock()
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	job, ok := s.jobs[id]
	if ok {
		delete(s.jobs, id)
	}
	s.mu.Unlock()
	if ok && job.FilePath != "" {
		_ = os.Remove(job.FilePath)
	}
	return ok
}

func (s *Store) StartWorkers() {
	for i := 0; i < s.cfg.MaxWorkers; i++ {
		go func() {
			for id := range s.queue {
				s.runJob(id)
			}
		}()
	}
}

func (s *Store) CleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.Cleanup(time.Now().UTC())
	}
}

func (s *Store) Cleanup(now time.Time) {
	var remove []string
	s.mu.Lock()
	for id, job := range s.jobs {
		if job.ExpiresAt != nil && now.After(*job.ExpiresAt) {
			remove = append(remove, id)
			continue
		}
		if job.State == StateFailed && now.Sub(job.UpdatedAt) > s.cfg.FailedTTL {
			remove = append(remove, id)
		}
	}
	s.mu.Unlock()
	for _, id := range remove {
		s.Delete(id)
	}
}

func (s *Store) runJob(id string) {
	job, ok := s.Get(id)
	if !ok {
		return
	}
	s.setState(id, StateProcessing, 0.2, "Muxing streams", nil)
	outDir := filepath.Join(s.cfg.WorkDir, id)
	filePath, err := s.cfg.Runner.Mux(context.Background(), media.JobRequest{
		VideoID:         job.VideoID,
		CanonicalURL:    job.CanonicalURL,
		VideoFormatID:   job.VideoFormat,
		AudioFormatID:   job.AudioFormat,
		OutputDirectory: outDir,
		OutputName:      "output.mp4",
	})
	if err != nil {
		s.setState(id, StateFailed, 1, "Failed", map[string]string{"code": "media_processing_failed", "message": "Media processing failed."})
		return
	}
	expiresAt := time.Now().UTC().Add(s.cfg.CompletedTTL)
	s.mu.Lock()
	if current, ok := s.jobs[id]; ok {
		current.State = StateCompleted
		current.FilePath = filePath
		current.ExpiresAt = &expiresAt
		current.UpdatedAt = time.Now().UTC()
	}
	s.mu.Unlock()
	s.publishByID(id, 1, "Completed", nil)
}

func (s *Store) setState(id string, state State, progress float64, message string, err interface{}) {
	s.mu.Lock()
	job, ok := s.jobs[id]
	if ok {
		job.State = state
		job.UpdatedAt = time.Now().UTC()
		if state == StateFailed {
			expires := time.Now().UTC().Add(s.cfg.FailedTTL)
			job.ExpiresAt = &expires
		}
	}
	s.mu.Unlock()
	if ok {
		s.publishByID(id, progress, message, err)
	}
}

func (s *Store) publishByID(id string, progress float64, message string, err interface{}) {
	s.mu.Lock()
	job, ok := s.jobs[id]
	s.mu.Unlock()
	if ok {
		s.publish(job, progress, message, err)
	}
}

func (s *Store) publish(job *Job, progress float64, message string, err interface{}) {
	s.mu.Lock()
	current, ok := s.jobs[job.ID]
	if !ok {
		s.mu.Unlock()
		return
	}
	event := Event{
		JobID:       current.ID,
		State:       current.State,
		Progress:    progress,
		Message:     message,
		DownloadURL: "",
		ExpiresAt:   current.ExpiresAt,
		Error:       err,
	}
	if current.State == StateCompleted {
		event.DownloadURL = "/api/jobs/" + current.ID + "/download"
	}
	current.events = append(current.events, event)
	subs := make([]chan Event, 0, len(current.subscribers))
	for ch := range current.subscribers {
		subs = append(subs, ch)
	}
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}
