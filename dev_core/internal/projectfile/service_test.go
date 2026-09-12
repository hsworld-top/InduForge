package projectfile

import "testing"

func TestValidateName(t *testing.T) {
	if got, err := validateName("video.mp4"); err != nil || got != "video.mp4" {
		t.Fatalf("valid file name rejected: %q %v", got, err)
	}
	for _, value := range []string{"", "../video.mp4", "a/b.mp4", "line\nfeed"} {
		if _, err := validateName(value); err == nil {
			t.Fatalf("invalid file name accepted: %q", value)
		}
	}
}

func TestValidatePath(t *testing.T) {
	if got, err := validatePath("/videos/demo/"); err != nil || got != "videos/demo" {
		t.Fatalf("path was not normalized: %q %v", got, err)
	}
	for _, value := range []string{"../videos", "videos/../private", `videos\\private`} {
		if _, err := validatePath(value); err == nil {
			t.Fatalf("invalid path accepted: %q", value)
		}
	}
}

func TestNormalizeContentType(t *testing.T) {
	if got := normalizeContentType("", "demo.mp4"); got != "video/mp4" {
		t.Fatalf("extension detection failed: %q", got)
	}
	if got := normalizeContentType("video/mp4; charset=utf-8", "demo.bin"); got != "video/mp4" {
		t.Fatalf("content type normalization failed: %q", got)
	}
}
