package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerSeparatesLivenessAndReadiness(t *testing.T) {
	state := &State{}
	handler := state.Handler()

	assertStatus := func(path string, want int) {
		t.Helper()
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != want {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, want)
		}
	}

	// 未就绪只影响流量接入，不影响进程存活判断。
	assertStatus("/health", http.StatusOK)
	assertStatus("/ready", http.StatusServiceUnavailable)

	state.SetReady(true)
	assertStatus("/health", http.StatusOK)
	assertStatus("/ready", http.StatusOK)
}
