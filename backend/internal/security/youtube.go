package security

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	errNoVideoID   = errors.New("a valid YouTube video ID is required")
)

type CanonicalVideo struct {
	ID  string `json:"videoId"`
	URL string `json:"canonicalUrl"`
}

func ParseYouTubeURL(raw string) (CanonicalVideo, error) {
	candidates := extractURLCandidates(raw)
	for _, candidate := range candidates {
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		id, ok := videoIDFromURL(parsed)
		if !ok {
			continue
		}
		return CanonicalVideo{
			ID:  id,
			URL: "https://www.youtube.com/watch?v=" + id,
		}, nil
	}
	return CanonicalVideo{}, errNoVideoID
}

func IsVideoID(value string) bool {
	return videoIDPattern.MatchString(value)
}

func CanonicalURLFromID(id string) (CanonicalVideo, error) {
	if !IsVideoID(id) {
		return CanonicalVideo{}, errNoVideoID
	}
	return CanonicalVideo{ID: id, URL: "https://www.youtube.com/watch?v=" + id}, nil
}

func extractURLCandidates(raw string) []string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, "<>()[]{}\"'")
		if strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://") {
			out = append(out, field)
		}
	}
	return out
}

func videoIDFromURL(u *url.URL) (string, bool) {
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", false
	}
	if u.Port() != "" {
		return "", false
	}

	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, ".") {
		return "", false
	}
	path := strings.Trim(u.EscapedPath(), "/")
	parts := strings.Split(path, "/")

	switch host {
	case "youtu.be":
		if len(parts) >= 1 && videoIDPattern.MatchString(parts[0]) {
			return parts[0], true
		}
	case "www.youtube.com", "youtube.com", "m.youtube.com", "music.youtube.com":
		if id := u.Query().Get("v"); videoIDPattern.MatchString(id) {
			return id, true
		}
		if len(parts) >= 2 && (parts[0] == "embed" || parts[0] == "shorts" || parts[0] == "live") && videoIDPattern.MatchString(parts[1]) {
			return parts[1], true
		}
	case "www.youtube-nocookie.com", "youtube-nocookie.com":
		if len(parts) >= 2 && parts[0] == "embed" && videoIDPattern.MatchString(parts[1]) {
			return parts[1], true
		}
	}

	return "", false
}
