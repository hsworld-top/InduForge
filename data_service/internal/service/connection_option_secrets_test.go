package service

import "testing"

func TestValidateConnectionOptionsContainNoSecrets(t *testing.T) {
	valid := map[string]any{"poolSize": 8, "compression": "zstd", "tls": map[string]any{"skipVerify": false}}
	if err := validateConnectionOptionsContainNoSecrets(valid); err != nil {
		t.Fatalf("普通扩展配置不应被拒绝: %v", err)
	}

	invalid := []map[string]any{
		{"password": "plain"},
		{"connection": map[string]any{"dsn": "ws://user:password@host"}},
		{"tls": map[string]any{"privateKey": "plain"}},
		{"headers": []any{map[string]any{"authorization": "Bearer plain"}}},
	}
	for _, options := range invalid {
		if err := validateConnectionOptionsContainNoSecrets(options); err == nil {
			t.Fatalf("敏感 options 应被拒绝: %#v", options)
		}
	}
}
