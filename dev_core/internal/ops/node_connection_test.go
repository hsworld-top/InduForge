package ops

import "testing"

func TestNodeConnectionURL(t *testing.T) {
	for _, value := range []string{"", "http://172.16.125.129:18080/", "https://center.example.com", "http://[fd00::1]:18080"} {
		h := &Handler{}
		if err := h.SetNodeConnectionURL(value); err != nil {
			t.Errorf("%s: %v", value, err)
		}
	}
	for _, value := range []string{"http://localhost:18080", "http://127.0.0.1", "http://0.0.0.0", "http://[::1]", "https://user:pass@center", "https://center/path", "https://center?token=x", "ftp://center"} {
		if err := (&Handler{}).SetNodeConnectionURL(value); err == nil {
			t.Errorf("accepted %s", value)
		}
	}
}
