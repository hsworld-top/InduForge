package ops

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
				var body struct {
					Data       map[string]string `json:"data"`
					StringData map[string]string `json:"stringData"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode patch body: %v", err)
				}
				if len(body.StringData) != 0 || body.Data["token"] == "" {
					t.Fatal("Secret SSA must use encoded data rather than stringData")
				}
				value, err := base64.StdEncoding.DecodeString(body.Data["token"])
				if err != nil || string(value) != "sensitive-value" {
					t.Fatal("Secret SSA data was not losslessly encoded")
				}
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
