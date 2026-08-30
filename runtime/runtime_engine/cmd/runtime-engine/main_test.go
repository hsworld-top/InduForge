package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewServerAppliesBoundedHTTPSettings(t *testing.T) {
	server := newServer(":0", http.NewServeMux())
	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 15*time.Second || server.WriteTimeout != 15*time.Second || server.IdleTimeout != 60*time.Second || server.MaxHeaderBytes != 1<<20 {
		t.Fatalf("unexpected server bounds: %+v", server)
	}
}
