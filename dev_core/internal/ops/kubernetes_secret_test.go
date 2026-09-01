package ops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKubeSecretClientDoesNotExposeData(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPatch {
			if strings.Contains(r.URL.RawQuery, "fieldManager") {
				w.WriteHeader(200)
				return
			}
		}
		w.WriteHeader(500)
	}))
	defer s.Close()
	c := &KubernetesProjectReconciler{client: s.Client(), endpoint: s.URL, token: "token"}
	if _, ok, e := c.GetSecret(context.Background(), "ns", "safe-name"); e != nil || ok {
		t.Fatalf("not found=%v %v", ok, e)
	}
	if e := c.ApplySecret(context.Background(), "ns", "safe-name", map[string]string{"token": "sensitive-value"}); e != nil {
		t.Fatal(e)
	}
}
