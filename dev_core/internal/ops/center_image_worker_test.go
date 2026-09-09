package ops

import (
	"context"
	"errors"
	"github.com/indu-forge/dev_core/internal/imagecatalog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCenterImageWorkerDoesNotBlockOrDuplicateAndUsesProcessContext(t *testing.T) {
	dir, e := os.MkdirTemp("", "if-iw-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "h.sock")
	t.Setenv("IF_OPS_CENTER_HOSTD_SOCKET", socket)
	l, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
			_, _ = w.Write([]byte(`{"code":0,"data":{"ready":true}}`))
		case <-r.Context().Done():
		}
	})}
	go server.Serve(l)
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := &PostgreSQLRepository{}
	repo.SetImageWorkerContext(ctx)
	n := Node{ID: "node"}
	artifacts := []imagecatalog.Artifact{{SHA256: "test", Size: 1}}
	assertPending := func() {
		t.Helper()
		start := time.Now()
		var pending *imagePreparationPending
		if err := repo.ensureCenterImagesAsync(n, artifacts); !errors.As(err, &pending) {
			t.Fatalf("expected pending: %v", err)
		}
		if time.Since(start) > time.Second {
			t.Fatal("worker blocked reconciliation")
		}
	}
	assertPending()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("worker not started")
	}
	assertPending()
	repo.centerImageWorkers.Lock()
	if len(repo.centerImageWorkers.slots) != 1 {
		t.Error("duplicate worker")
	}
	repo.centerImageWorkers.Unlock()
	close(release)
	deadline := time.Now().Add(time.Second)
	for {
		repo.centerImageWorkers.Lock()
		running := repo.centerImageWorkers.nodes[n.ID].running
		repo.centerImageWorkers.Unlock()
		if !running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not complete")
		}
		time.Sleep(time.Millisecond)
	}
	if e = repo.ensureCenterImagesAsync(n, artifacts); e != nil {
		t.Fatal(e)
	}
	cancel()
	if e = repo.ensureCenterImagesAsync(n, artifacts); !errors.Is(e, context.Canceled) {
		t.Fatalf("process shutdown ignored: %v", e)
	}
}
