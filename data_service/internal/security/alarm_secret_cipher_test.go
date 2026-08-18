package security

import (
	"bytes"
	"testing"
)

func TestAlarmSecretCipherRoundTrip(t *testing.T) {
	cipher, err := NewAlarmSecretCipher(bytes.Repeat([]byte{7}, 32), "v1")
	if err != nil {
		t.Fatalf("NewAlarmSecretCipher() error = %v", err)
	}
	payload, err := cipher.Encrypt([]byte("secret-value"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if bytes.Contains(payload, []byte("secret-value")) {
		t.Fatal("ciphertext contains plaintext")
	}
	plaintext, err := cipher.Decrypt(payload)
	if err != nil || string(plaintext) != "secret-value" {
		t.Fatalf("Decrypt() = %q, %v", plaintext, err)
	}
	payload[len(payload)-1] ^= 1
	if _, err = cipher.Decrypt(payload); err == nil {
		t.Fatal("tampered ciphertext should be rejected")
	}
}
