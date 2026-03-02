package infra

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
)

// DeviceIdentity holds Ed25519 keypair and derived device ID
type DeviceIdentity struct {
	DeviceID     string
	PublicKeyRaw []byte // raw 32-byte Ed25519 public key
	PrivateKey   ed25519.PrivateKey
}

// GenerateDeviceIdentity generates a new device identity (no file I/O; suitable for embedded)
func GenerateDeviceIdentity() (*DeviceIdentity, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	return &DeviceIdentity{
		DeviceID:     fingerprintPublicKey(pub),
		PublicKeyRaw: pub,
		PrivateKey:   priv,
	}, nil
}

func fingerprintPublicKey(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return encodeHex(h[:])
}

func encodeHex(b []byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hex[v>>4]
		out[i*2+1] = hex[v&0xf]
	}
	return string(out)
}

// SignDevicePayload signs the auth payload string with Ed25519, returns base64url signature
func SignDevicePayload(ident *DeviceIdentity, payload string) string {
	sig := ed25519.Sign(ident.PrivateKey, []byte(payload))
	return base64URLEncode(sig)
}

// PublicKeyBase64URL returns raw public key as base64url
func PublicKeyBase64URL(ident *DeviceIdentity) string {
	return base64URLEncode(ident.PublicKeyRaw)
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
