package auth

import (
	"strings"
	"testing"
)

func TestDeriveKey_Deterministic(t *testing.T) {
	key1 := DeriveKey("my-secret")
	key2 := DeriveKey("my-secret")
	if string(key1) != string(key2) {
		t.Error("DeriveKey should be deterministic for same input")
	}
}

func TestDeriveKey_DifferentInputs(t *testing.T) {
	key1 := DeriveKey("secret-a")
	key2 := DeriveKey("secret-b")
	if string(key1) == string(key2) {
		t.Error("DeriveKey should produce different keys for different inputs")
	}
}

func TestDeriveKey_Length(t *testing.T) {
	key := DeriveKey("test")
	if len(key) != 32 {
		t.Errorf("DeriveKey should produce 32-byte key, got %d", len(key))
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := DeriveKey("master-key")
	plaintext := "my-secret-token-12345"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == plaintext {
		t.Error("Encrypted value should differ from plaintext")
	}

	if !IsEncrypted(encrypted) {
		t.Error("Encrypted value should have enc:v1: prefix")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecrypt_PlaintextPassthrough(t *testing.T) {
	key := DeriveKey("any-key")
	plaintext := "not-encrypted-value"

	result, err := Decrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Decrypt of plaintext should not error: %v", err)
	}
	if result != plaintext {
		t.Errorf("Plaintext passthrough failed: got %q, want %q", result, plaintext)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := DeriveKey("correct-key")
	key2 := DeriveKey("wrong-key")

	encrypted, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"enc:v1:abcdef", true},
		{"enc:v1:", true},
		{"plaintext", false},
		{"enc:v2:something", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := IsEncrypted(tc.value); got != tc.expected {
			t.Errorf("IsEncrypted(%q) = %v, want %v", tc.value, got, tc.expected)
		}
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := DeriveKey("key")
	plaintext := "same-value"

	enc1, _ := Encrypt(plaintext, key)
	enc2, _ := Encrypt(plaintext, key)

	if enc1 == enc2 {
		t.Error("Two encryptions of same plaintext should differ (unique nonces)")
	}
}

func TestEncrypt_EmptyString(t *testing.T) {
	key := DeriveKey("key")

	encrypted, err := Encrypt("", key)
	if err != nil {
		t.Fatalf("Encrypt empty string failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt empty string failed: %v", err)
	}
	if decrypted != "" {
		t.Errorf("Expected empty string, got %q", decrypted)
	}
}

func TestGetMasterKey_EnvVar(t *testing.T) {
	t.Setenv("PICOCLAW_MASTER_KEY", "test-master")
	if got := GetMasterKey(); got != "test-master" {
		t.Errorf("GetMasterKey() = %q, want %q", got, "test-master")
	}
}

func TestGetMasterKey_Unset(t *testing.T) {
	t.Setenv("PICOCLAW_MASTER_KEY", "")
	if got := GetMasterKey(); got != "" {
		t.Errorf("GetMasterKey() should be empty when env unset, got %q", got)
	}
}

func TestEncryptedPrefix(t *testing.T) {
	key := DeriveKey("key")
	encrypted, _ := Encrypt("test", key)
	if !strings.HasPrefix(encrypted, "enc:v1:") {
		t.Errorf("Encrypted value should start with 'enc:v1:', got %q", encrypted)
	}
}
