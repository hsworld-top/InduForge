package security

import (
	"bytes"
	"testing"
)

func TestCollectorSecretCipherRoundTripUsesRandomNonce(t *testing.T) {
	secretCipher, err := NewCollectorSecretCipher(bytes.Repeat([]byte{1}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	first, err := secretCipher.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := secretCipher.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("ciphertext must use a random nonce")
	}
	plaintext, err := secretCipher.Decrypt(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "secret" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
	if secretCipher.KeyVersion() != "v1" {
		t.Fatalf("unexpected key version %q", secretCipher.KeyVersion())
	}
}

func TestCollectorSecretCipherRejectsTamperedPayload(t *testing.T) {
	secretCipher, err := NewCollectorSecretCipher(bytes.Repeat([]byte{2}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := secretCipher.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	payload[len(payload)-1] ^= 0xff
	if _, err := secretCipher.Decrypt(payload); err == nil {
		t.Fatal("expected tampered payload to be rejected")
	}
}

func TestNewCollectorSecretCipherRejectsInvalidKey(t *testing.T) {
	if _, err := NewCollectorSecretCipher([]byte("short"), "v1"); err == nil {
		t.Fatal("expected invalid key length")
	}
	if _, err := NewCollectorSecretCipher(bytes.Repeat([]byte{1}, 32), ""); err == nil {
		t.Fatal("expected empty key version")
	}
}
