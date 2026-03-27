package zalo

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

type PKCEPair struct{ Verifier, Challenge string }

func GeneratePKCE() (PKCEPair, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return PKCEPair{}, err
	}
	v := base64.RawURLEncoding.EncodeToString(raw)
	h := sha256.Sum256([]byte(v))
	return PKCEPair{Verifier: v, Challenge: base64.RawURLEncoding.EncodeToString(h[:])}, nil
}
