package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

type PKCECodes struct {
	CodeVerifier  string
	CodeChallenge string
}

func GeneratePKCE() (PKCECodes, error) {
	buf := make([]byte, 64)
	if _, err := rand.Read(buf); err != nil {
		return PKCECodes{}, err
	}

	verifier := base64.RawURLEncoding.EncodeToString(buf)

	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return PKCECodes{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}, nil
}
