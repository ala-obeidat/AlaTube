package security

import "testing"

func TestParseYouTubeURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		id   string
		ok   bool
	}{
		{"watch", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=abc", "dQw4w9WgXcQ", true},
		{"short", "share https://youtu.be/a_b-CdEf123?si=x", "a_b-CdEf123", true},
		{"embed", "https://www.youtube-nocookie.com/embed/a_b-CdEf123", "a_b-CdEf123", true},
		{"playlist without video", "https://www.youtube.com/playlist?list=PLabc", "", false},
		{"bad host", "https://example.com/watch?v=dQw4w9WgXcQ", "", false},
		{"bad id chars", "https://www.youtube.com/watch?v=dQw4w9WgXc!", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseYouTubeURL(tt.raw)
			if tt.ok && err != nil {
				t.Fatalf("expected ok: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("expected error, got %#v", got)
			}
			if got.ID != tt.id {
				t.Fatalf("id = %q, want %q", got.ID, tt.id)
			}
		})
	}
}

