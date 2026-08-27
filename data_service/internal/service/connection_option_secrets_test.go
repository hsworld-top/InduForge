package service

import "testing"

func TestNormalizeConnectionSecretStatus(t *testing.T) {
	if status := normalizeConnectionSecretStatus(nil); status == nil || len(status) != 0 {
		t.Fatalf("nil 密钥状态应归一为空对象，实际 %#v", status)
	}
	original := map[string]bool{"password": true}
	if status := normalizeConnectionSecretStatus(original); !status["password"] {
		t.Fatalf("已有密钥状态不应被改写，实际 %#v", status)
	}
}

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

func TestExtractGenericConnectionSecretsSeparatesTLSPEM(t *testing.T) {
	secrets, clearKeys, config := extractGenericConnectionSecrets(map[string]any{
		"dbType":   "postgresql",
		"password": "database-secret",
		"sslConfig": map[string]any{
			"mode": "verify-full",
			"ca":   "ca-pem",
			"cert": "client-pem",
			"key":  "private-key-pem",
		},
	})
	if len(clearKeys) != 0 {
		t.Fatalf("non-empty secrets should not be cleared: %#v", clearKeys)
	}
	for key, expected := range map[string]string{
		"password": "database-secret", "tls.ca": "ca-pem",
		"tls.cert": "client-pem", "tls.key": "private-key-pem",
	} {
		if secrets[key] != expected {
			t.Fatalf("secret %s = %q, expected %q", key, secrets[key], expected)
		}
	}
	sslConfig := mapFromAny(config["sslConfig"])
	if sslConfig["mode"] != "verify-full" || sslConfig["ca"] != nil || sslConfig["key"] != nil {
		t.Fatalf("metadata should only keep non-sensitive TLS fields: %#v", sslConfig)
	}
}

func TestExtractGenericConnectionSecretsKeepsMissingTLSPEM(t *testing.T) {
	_, clearKeys, config := extractGenericConnectionSecrets(map[string]any{
		"dbType": "mysql", "sslConfig": map[string]any{"mode": "require"},
	})
	if len(clearKeys) != 0 {
		t.Fatalf("omitted TLS fields must keep saved secrets: %#v", clearKeys)
	}
	if mapFromAny(config["sslConfig"])["mode"] != "require" {
		t.Fatalf("TLS mode should remain in metadata: %#v", config)
	}
}
