package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func TestAbortIsNilSafeConcurrentAndClosesGate(t *testing.T) {
	var nilStore *Store
	nilStore.Abort()
	store := &Store{}
	var group sync.WaitGroup
	for range 16 {
		group.Add(1)
		go func() { defer group.Done(); store.Abort() }()
	}
	group.Wait()
	for _, err := range []error{store.Ping(context.Background()), store.VerifySchema(context.Background()), store.ApplySchema(context.Background())} {
		if !errors.Is(err, ErrStoreClosed) {
			t.Fatalf("closed store error=%v, want ErrStoreClosed", err)
		}
	}
	store.Close()
}

// This test uses only an explicitly supplied PostgreSQL and holds one pool
// connection to prove Abort itself never waits for pgxpool.Close completion.
func TestAbortReturnsBeforeHeldConnectionAndRejectsNewWork(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, _, cleanup := setupIntegrationStore(t, ctx, dsn)
	defer cleanup()
	store.closeDone = make(chan struct{})
	conn, err := store.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	store.Abort()
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Fatalf("Abort blocked on held connection: %s", elapsed)
	}
	if err := store.Ping(ctx); !errors.Is(err, ErrStoreClosed) {
		t.Fatalf("new work after Abort error=%v", err)
	}
	select {
	case <-store.closeDone:
		t.Fatal("background pool close completed before held connection released")
	default:
	}
	conn.Release()
	select {
	case <-store.closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("background pool close did not complete after release")
	}
}
