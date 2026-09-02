package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckLoopbackHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	if err := checkLoopbackHealth(server.URL+"/health", server.Client()); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"http://example.com:18081/health", server.URL + "/other", "https://127.0.0.1:18081/health"} {
		if err := checkLoopbackHealth(endpoint, server.Client()); err == nil {
			t.Fatalf("unsafe endpoint accepted: %s", endpoint)
		}
	}
}

func TestCheckLoopbackHealthRejectsTimeoutAndBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if err := checkLoopbackHealth(server.URL+"/health", server.Client()); err == nil {
		t.Fatal("bad status must be rejected")
	}
	timeoutClient := server.Client()
	timeoutClient.Timeout = time.Nanosecond
	if err := checkLoopbackHealth(server.URL+"/health", timeoutClient); err == nil {
		t.Fatal("timeout must be rejected")
	}
}

func TestLoadNATSTokenRequiresStrictTokenCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"project-token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	token, err := loadNATSToken(path)
	if err != nil || token != "project-token" {
		t.Fatalf("token=%q err=%v", token, err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"project-token","extra":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadNATSToken(path); err == nil {
		t.Fatal("unknown field must be rejected")
	}
}
