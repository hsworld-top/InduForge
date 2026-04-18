package config

import (
	"testing"
)

func TestLoad_UsesDefaultAddr(t *testing.T) {
	t.Setenv("DATA_SERVICE_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
}

func TestLoad_RejectsInvalidAddr(t *testing.T) {
	t.Setenv("DATA_SERVICE_ADDR", "invalid-addr")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
