package auth

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuthorizerAcceptsHashedBearerAndRejectsExpired(t *testing.T) {
	now := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	validDigest := DigestToken("valid-token")
	expiredDigest := DigestToken("expired-token")
	payload := fmt.Sprintf(`{"schemaVersion":"runtime-api-tokens.v1","tokens":[{"tokenSha256":"%s","subjectId":"user-1","roles":["viewer"]},{"tokenSha256":"%s","subjectId":"user-2","roles":["viewer"],"expiresAt":"2026-08-31T09:00:00Z"}]}`, validDigest, expiredDigest)
	path := filepath.Join(t.TempDir(), "tokens.json")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	authorizer, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	principal, err := authorizer.Authenticate(request, now)
	if err != nil || principal.SubjectID != "user-1" {
		t.Fatalf("unexpected principal=%#v err=%v", principal, err)
	}
	request.Header.Set("Authorization", "Bearer expired-token")
	if _, err := authorizer.Authenticate(request, now); err == nil {
		t.Fatal("expired token must be rejected")
	}
}
