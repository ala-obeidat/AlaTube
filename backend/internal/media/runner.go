package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Format struct {
	FormatID       string `json:"formatId"`
	Kind           string `json:"kind"`
	Height         int    `json:"height,omitempty"`
	FPS            int    `json:"fps,omitempty"`
	Container      string `json:"container,omitempty"`
	Codec          string `json:"codec,omitempty"`
	EstimatedBytes int64  `json:"estimatedBytes,omitempty"`
}

type Analysis struct {
	VideoID         string   `json:"videoId"`
	CanonicalURL    string   `json:"canonicalUrl"`
	Title           string   `json:"title"`
	DurationSeconds int      `json:"durationSeconds"`
	ThumbnailURL    string   `json:"thumbnailUrl,omitempty"`
	Formats         []Format `json:"formats"`
}

type JobRequest struct {
	VideoID         string
	CanonicalURL    string
	VideoFormatID   string
	AudioFormatID   string
	OutputDirectory string
	OutputName      string
}

type Runner interface {
	Analyze(ctx context.Context, canonicalURL, videoID string) (Analysis, error)
	Mux(ctx context.Context, req JobRequest) (string, error)
}

type RunnerConfig struct {
	Image       string
	WorkDir     string
	JobTimeout  time.Duration
	MemoryLimit string
	CPUs        string
}

type DockerRunner struct {
	cfg RunnerConfig
}

func NewDockerRunner(cfg RunnerConfig) *DockerRunner {
	return &DockerRunner{cfg: cfg}
}

func (r *DockerRunner) Analyze(ctx context.Context, canonicalURL, videoID string) (Analysis, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.JobTimeout)
	defer cancel()

	args := r.baseDockerArgs()
	args = append(args, r.cfg.Image, "yt-dlp", "--dump-json", "--no-playlist", canonicalURL)
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		return Analysis{}, fmt.Errorf("media analysis failed")
	}

	var payload struct {
		Title      string `json:"title"`
		Duration   int    `json:"duration"`
		Thumbnail  string `json:"thumbnail"`
		FormatsRaw []struct {
			FormatID string `json:"format_id"`
			VCodec   string `json:"vcodec"`
			ACodec   string `json:"acodec"`
			Ext      string `json:"ext"`
			Height   int    `json:"height"`
			FPS      int    `json:"fps"`
			Filesize int64  `json:"filesize"`
			Approx   int64  `json:"filesize_approx"`
		} `json:"formats"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return Analysis{}, errors.New("invalid media analysis response")
	}

	formats := make([]Format, 0, len(payload.FormatsRaw))
	for _, f := range payload.FormatsRaw {
		kind := "muxed"
		codec := f.VCodec
		if f.VCodec == "none" {
			kind = "audio"
			codec = f.ACodec
		} else if f.ACodec == "none" {
			kind = "video"
		}
		size := f.Filesize
		if size == 0 {
			size = f.Approx
		}
		formats = append(formats, Format{
			FormatID:       f.FormatID,
			Kind:           kind,
			Height:         f.Height,
			FPS:            f.FPS,
			Container:      f.Ext,
			Codec:          codec,
			EstimatedBytes: size,
		})
	}

	return Analysis{
		VideoID:         videoID,
		CanonicalURL:    canonicalURL,
		Title:           payload.Title,
		DurationSeconds: payload.Duration,
		ThumbnailURL:    payload.Thumbnail,
		Formats:         formats,
	}, nil
}

func (r *DockerRunner) Mux(ctx context.Context, req JobRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.JobTimeout)
	defer cancel()

	if err := os.MkdirAll(req.OutputDirectory, 0o700); err != nil {
		return "", err
	}

	outputPath := filepath.Join(req.OutputDirectory, req.OutputName)
	format := req.VideoFormatID
	if req.AudioFormatID != "" {
		format += "+" + req.AudioFormatID
	}

	args := r.baseDockerArgs()
	args = append(args,
		"--mount", "type=bind,src="+req.OutputDirectory+",dst=/work",
		r.cfg.Image,
		"yt-dlp",
		"--no-playlist",
		"--format", format,
		"--merge-output-format", "mp4",
		"--output", "/work/output.%(ext)s",
		req.CanonicalURL,
	)
	cmd := exec.CommandContext(ctx, "docker", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("media mux failed")
	}
	return outputPath, nil
}

func (r *DockerRunner) baseDockerArgs() []string {
	return []string{
		"run", "--rm",
		"--network", "alatube_media",
		"--user", "10001:10001",
		"--read-only",
		"--security-opt", "no-new-privileges:true",
		"--cap-drop", "ALL",
		"--cpus", r.cfg.CPUs,
		"--memory", r.cfg.MemoryLimit,
		"--pids-limit", "128",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=128m",
	}
}

