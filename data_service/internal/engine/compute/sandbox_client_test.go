package compute

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSandboxClientPropagatesRequestCancellation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
		close(cancelled)
	}))
	defer server.Close()

	client := NewSandboxClient(server.URL, "test-token")
	ctx, cancel := context.WithCancel(context.Background())
	errResult := make(chan error, 1)
	go func() {
		_, err := client.Capabilities(ctx)
		errResult <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("sandbox request did not start")
	}
	cancel()
	if err := <-errResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("sandbox server did not observe request cancellation")
	}
}
