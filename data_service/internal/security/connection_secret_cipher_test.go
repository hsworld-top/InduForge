package security

import (
	"bytes"
	"testing"
)

func TestConnectionSecretCipherBindsCiphertextToConnectionAndKey(t *testing.T) {
	secretCipher, err := NewConnectionSecretCipher(bytes.Repeat([]byte{3}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := secretCipher.Encrypt("connection-a", "password", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := secretCipher.Decrypt("connection-a", "password", payload)
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
	if _, err := secretCipher.Decrypt("connection-b", "password", payload); err == nil {
		t.Fatal("expected ciphertext to be rejected for another connection")
	}
	if _, err := secretCipher.Decrypt("connection-a", "token", payload); err == nil {
		t.Fatal("expected ciphertext to be rejected for another secret key")
	}
}

func TestConnectionSecretCipherUsesRandomNonceAndRejectsInvalidConfig(t *testing.T) {
	secretCipher, err := NewConnectionSecretCipher(bytes.Repeat([]byte{4}, 32), "v2")
	if err != nil {
		t.Fatal(err)
	}
	first, err := secretCipher.Encrypt("connection-a", "password", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := secretCipher.Encrypt("connection-a", "password", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("ciphertext must use a random nonce")
	}
	if secretCipher.KeyVersion() != "v2" {
		t.Fatalf("unexpected key version %q", secretCipher.KeyVersion())
	}
	if _, err := NewConnectionSecretCipher([]byte("short"), "v1"); err == nil {
		t.Fatal("expected invalid key length")
	}
	if _, err := NewConnectionSecretCipher(bytes.Repeat([]byte{1}, 32), ""); err == nil {
		t.Fatal("expected empty key version")
	}
}
