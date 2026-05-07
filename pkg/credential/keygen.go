package credential

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// DefaultSSHKeyPath returns the canonical path for the picoclaw-specific SSH key.
// The path is always ~/.ssh/picoclaw_ed25519.key (os.UserHomeDir is cross-platform).
func DefaultSSHKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("credential: cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "picoclaw_ed25519.key"), nil
}

// GenerateSSHKey generates an Ed25519 SSH key pair and writes the private key
// to path (permissions 0600) and the public key to path+".pub" (permissions 0644).
// The ~/.ssh/ directory is created with 0700 if it does not exist.
// If the files already exist they are overwritten.
func GenerateSSHKey(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("credential: keygen: cannot create directory %q: %w", filepath.Dir(path), err)
	}

	pubRaw, privRaw, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("credential: keygen: ed25519 key generation failed: %w", err)
	}

	// Marshal private key as OpenSSH PEM.
	block, err := ssh.MarshalPrivateKey(privRaw, "")
	if err != nil {
		return fmt.Errorf("credential: keygen: marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(block)

	if err = os.WriteFile(path, privPEM, 0o600); err != nil {
		return fmt.Errorf("credential: keygen: write private key %q: %w", path, err)
	}

	// Marshal public key as authorized_keys line.
	sshPub, err := ssh.NewPublicKey(pubRaw)
	if err != nil {
		return fmt.Errorf("credential: keygen: marshal public key: %w", err)
	}
	pubLine := ssh.MarshalAuthorizedKey(sshPub)

	pubPath := path + ".pub"
	if err := os.WriteFile(pubPath, pubLine, 0o644); err != nil {
		return fmt.Errorf("credential: keygen: write public key %q: %w", pubPath, err)
	}

	return nil
}
