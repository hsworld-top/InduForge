package service

import "testing"

func TestWorkbenchSecretsAreMaskedAndHydrated(t *testing.T) {
	headers, auth, secrets := splitWorkbenchSecrets(
		[]any{
			map[string]any{"key": "Authorization", "value": "Bearer secret"},
			map[string]any{"key": "Accept", "value": "application/json"},
		},
		map[string]any{"type": "bearer", "token": "token-value", "password": "password-value"},
	)
	first := headers[0].(map[string]any)
	if first["value"] != "" || first["secretSet"] != true || auth["token"] != nil || auth["password"] != nil {
		t.Fatalf("masked payload is invalid: headers=%#v auth=%#v", headers, auth)
	}
	if len(secrets) != 3 {
		t.Fatalf("expected three encrypted values, got %#v", secrets)
	}
	hydratedHeaders, hydratedAuth := hydrateWorkbenchSecrets(headers, auth, secrets)
	if hydratedHeaders[0].(map[string]any)["value"] != "Bearer secret" || hydratedAuth["token"] != "token-value" || hydratedAuth["password"] != "password-value" {
		t.Fatalf("hydrated payload is invalid: headers=%#v auth=%#v", hydratedHeaders, hydratedAuth)
	}
}

func TestWorkbenchSensitiveHeaderDetection(t *testing.T) {
	for _, name := range []string{"Authorization", "X-API-Key", "Cookie", "x-private-token"} {
		if !isSensitiveWorkbenchHeader(name) {
			t.Fatalf("%s should be sensitive", name)
		}
	}
	if isSensitiveWorkbenchHeader("Content-Type") {
		t.Fatal("Content-Type should not be sensitive")
	}
}
