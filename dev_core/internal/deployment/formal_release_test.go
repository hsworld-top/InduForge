package deployment

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestAssembleFormalReleaseFailsWithoutRuntimeConfig(t *testing.T) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	_, err := assembleFormalRelease(Project{ID: "11111111-1111-4111-8111-111111111111", Code: "demo"}, Version{ID: "22222222-2222-4222-8222-222222222222", Version: "1.0.0"}, ReleaseSource{}, SigningConfig{Key: key, KeyID: "key"}, ServiceConfig{}, time.Now())
	if err == nil {
		t.Fatal("missing minimum versions must fail closed")
	}
}
