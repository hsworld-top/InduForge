package worker_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/worker"
)

func TestWorkerRecoversImmediatelyAndStops(t *testing.T) {
	repository := &fakeRepository{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.New(repository, fakeSigner{}, nil, time.Hour).Run(ctx)
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for repository.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if repository.calls.Load() == 0 {
		t.Fatal("Worker 启动后未立即恢复任务")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Worker 未随 context 停止")
	}
}

type fakeRepository struct {
	calls   atomic.Int32
	payload []byte
}

func (r *fakeRepository) ListStaleCommands(context.Context) ([]worker.StaleCommand, error) {
	payload, _ := json.Marshal(map[string]any{"artifactKey": "versions/test.ifp", "artifactUrl": "expired"})
	return []worker.StaleCommand{{ID: "11111111-1111-4111-8111-111111111111", Payload: payload, Attempts: 1, MaxAttempts: 3}}, nil
}

func (r *fakeRepository) RecoverStaleCommand(_ context.Context, _ string, payload []byte) (bool, error) {
	r.calls.Add(1)
	r.payload = payload
	return true, nil
}

type fakeSigner struct{}

func (fakeSigner) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "https://objects/signed?fresh=true", nil
}
