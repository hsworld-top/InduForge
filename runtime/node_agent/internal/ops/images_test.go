package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indu-forge/node_agent/internal/hostd"
)

func imageFixture(body string) ImageDownloadArtifact {
	sum := sha256.Sum256([]byte(body))
	return ImageDownloadArtifact{ImageArtifact: hostd.ImageArtifact{SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body)), Images: []hostd.ImageReference{{Reference: "redis:7", ConfigDigest: "sha256:" + strings.Repeat("a", 64)}}}, DownloadPath: "/image"}
}
func TestDownloadImageResumeAndReuse(t *testing.T) {
	body := "verified archive payload"
	artifact := imageFixture(body)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("Range") != "bytes=5-" {
			t.Errorf("unexpected headers: %v", r.Header)
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 5-%d/%d", len(body)-1, len(body)))
		w.WriteHeader(206)
		_, _ = w.Write([]byte(body[5:]))
	}))
	defer server.Close()
	a := &Agent{cfg: Config{ServerURL: server.URL}, client: server.Client(), identity: Identity{AgentToken: "token"}}
	dir := t.TempDir()
	target := filepath.Join(dir, artifact.SHA256+".tar")
	if err := os.WriteFile(target+".part", []byte(body[:5]), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.downloadImage(context.Background(), artifact, dir); err != nil {
		t.Fatal(err)
	}
	if err := a.downloadImage(context.Background(), artifact, dir); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !verifyImageFile(target, artifact) {
		t.Fatal("cache not reused or invalid")
	}
}
func TestDownloadImageRejectsCorruptionRangeAndRedirect(t *testing.T) {
	for _, kind := range []string{"corrupt", "range", "redirect"} {
		t.Run(kind, func(t *testing.T) {
			artifact := imageFixture("correct")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch kind {
				case "corrupt":
					_, _ = w.Write([]byte("invalid"))
				case "range":
					w.Header().Set("Content-Range", "bytes 3-6/7")
					w.WriteHeader(206)
					_, _ = w.Write([]byte("rect"))
				case "redirect":
					w.Header().Set("Location", "/secret")
					w.WriteHeader(302)
				}
			}))
			defer server.Close()
			a := &Agent{cfg: Config{ServerURL: server.URL}, client: server.Client()}
			dir := t.TempDir()
			if err := a.downloadImage(context.Background(), artifact, dir); err == nil {
				t.Fatal("invalid response accepted")
			}
			if _, err := os.Stat(filepath.Join(dir, artifact.SHA256+".tar")); !os.IsNotExist(err) {
				t.Fatal("invalid archive published")
			}
		})
	}
}
