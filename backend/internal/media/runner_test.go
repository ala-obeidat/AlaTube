package media

import "testing"

func TestClassifyFormat(t *testing.T) {
	cases := []struct {
		name, vcodec, acodec, wantKind, wantCodec string
	}{
		{"storyboard image (both none)", "none", "none", "image", ""},
		{"audio only", "none", "mp4a.40.2", "audio", "mp4a.40.2"},
		{"video only", "avc1.4d401f", "none", "video", "avc1.4d401f"},
		{"muxed", "avc1.42001E", "mp4a.40.2", "muxed", "avc1.42001E"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, codec := classifyFormat(c.vcodec, c.acodec)
			if kind != c.wantKind || codec != c.wantCodec {
				t.Fatalf("classifyFormat(%q,%q) = (%q,%q), want (%q,%q)", c.vcodec, c.acodec, kind, codec, c.wantKind, c.wantCodec)
			}
		})
	}
}
