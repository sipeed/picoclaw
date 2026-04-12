package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	codes, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	if codes.CodeVerifier == "" {
		t.Fatal("CodeVerifier is empty")
	}
	if codes.CodeChallenge == "" {
		t.Fatal("CodeChallenge is empty")
	}

	verifierBytes, err := base64.RawURLEncoding.DecodeString(codes.CodeVerifier)
	if err != nil {
		t.Fatalf("CodeVerifier is not valid base64url: %v", err)
	}
	if len(verifierBytes) != 64 {
		t.Errorf("CodeVerifier decoded length = %d, want 64", len(verifierBytes))
	}

	hash := sha256.Sum256([]byte(codes.CodeVerifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	if codes.CodeChallenge != expectedChallenge {
		t.Errorf("CodeChallenge = %q, want SHA256 of verifier = %q", codes.CodeChallenge, expectedChallenge)
	}
}

func TestGeneratePKCEUniqueness(t *testing.T) {
	codes1, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	codes2, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	if codes1.CodeVerifier == codes2.CodeVerifier {
		t.Error("two GeneratePKCE() calls produced identical verifiers")
	}
}
