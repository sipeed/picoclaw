package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryption(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
	}{
		{"ChaCha20-Poly1305", "chacha20-poly1305"},
		{"AES-256-GCM", "aes-256-gcm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for key
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			encryptor, err := NewEncryptor(tt.algorithm)
			require.NoError(t, err)
			require.NotNil(t, encryptor)

			plaintext := []byte("sensitive-api-key-12345")

			// Test encryption
			encData, err := encryptor.Encrypt(plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, encData.Ciphertext)
			assert.NotEmpty(t, encData.Nonce)
			assert.Equal(t, tt.algorithm, encData.Algorithm)

			// Test decryption
			decrypted, err := encryptor.Decrypt(encData)
			require.NoError(t, err)
			assert.Equal(t, plaintext, decrypted)
		})
	}
}

func TestEncryptionCredential(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	encryptor, err := NewEncryptor("chacha20-poly1305")
	require.NoError(t, err)

	cred := &AuthCredential{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		AccountID:    "test-account-id",
		ExpiresAt:    time.Now().Add(time.Hour),
		Provider:     "anthropic",
		AuthMethod:   "oauth",
		Email:        "test@example.com",
	}

	// Test encrypting credential
	encData, err := encryptor.EncryptCredential(cred)
	require.NoError(t, err)
	assert.NotEmpty(t, encData.Ciphertext)

	// Test decrypting credential
	decrypted, err := encryptor.DecryptCredential(encData)
	require.NoError(t, err)
	assert.Equal(t, cred.AccessToken, decrypted.AccessToken)
	assert.Equal(t, cred.RefreshToken, decrypted.RefreshToken)
	assert.Equal(t, cred.AccountID, decrypted.AccountID)
	assert.Equal(t, cred.Provider, decrypted.Provider)
	assert.Equal(t, cred.AuthMethod, decrypted.AuthMethod)
	assert.Equal(t, cred.Email, decrypted.Email)
}

func TestEncryptionInvalidAlgorithm(t *testing.T) {
	_, err := NewEncryptor("invalid-algorithm")
	assert.ErrorIs(t, err, ErrUnsupportedAlgorithm)
}

func TestEncryptionInvalidCiphertext(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	encryptor, err := NewEncryptor("chacha20-poly1305")
	require.NoError(t, err)

	// Test with invalid base64
	_, err = encryptor.Decrypt(&EncryptedData{
		Algorithm:  "chacha20-poly1305",
		Nonce:      "not-valid-base64!!!",
		Ciphertext: "YWJjZA==",
	})
	assert.Error(t, err)

	// Test with invalid nonce size
	_, err = encryptor.Decrypt(&EncryptedData{
		Algorithm:  "chacha20-poly1305",
		Nonce:      "YWJjZA==", // "abcd" - too short
		Ciphertext: "YWJjZA==",
	})
	assert.ErrorIs(t, err, ErrInvalidCiphertext)
}

func TestMockKeychain(t *testing.T) {
	keychain := NewMockKeychain()
	assert.True(t, keychain.IsAvailable())

	cred := &AuthCredential{
		AccessToken: "test-token",
		Provider:    "test-provider",
		AuthMethod:  "token",
	}

	// Test store
	err := keychain.Store("test-provider", cred)
	require.NoError(t, err)

	// Test retrieve
	retrieved, err := keychain.Retrieve("test-provider")
	require.NoError(t, err)
	assert.Equal(t, cred.AccessToken, retrieved.AccessToken)

	// Test retrieve non-existent
	retrieved, err = keychain.Retrieve("non-existent")
	require.NoError(t, err)
	assert.Nil(t, retrieved)

	// Test delete
	err = keychain.Delete("test-provider")
	require.NoError(t, err)

	retrieved, err = keychain.Retrieve("test-provider")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSecureStorePlain(t *testing.T) {
	ResetSecureStore()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Test with encryption disabled
	store, err := NewSecureStore(SecureStoreConfig{
		Enabled:     false,
		UseKeychain: false,
	})
	require.NoError(t, err)

	cred := &AuthCredential{
		AccessToken: "plain-text-token",
		Provider:    "test-provider",
		AuthMethod:  "token",
	}

	// Store credential
	err = store.SetCredential("test-provider", cred)
	require.NoError(t, err)

	// Retrieve credential
	retrieved, err := store.GetCredential("test-provider")
	require.NoError(t, err)
	assert.Equal(t, cred.AccessToken, retrieved.AccessToken)

	// Verify it's stored in plain text
	authFile := filepath.Join(tmpDir, ".picoclaw", "auth.json")
	data, err := os.ReadFile(authFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "plain-text-token")
}

func TestSecureStoreEncrypted(t *testing.T) {
	ResetSecureStore()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Use mock keychain for testing
	mockKeychain := NewMockKeychain()
	store := &SecureStore{
		config: SecureStoreConfig{
			Enabled:     true,
			UseKeychain: false,
			Algorithm:   "chacha20-poly1305",
		},
		keychain: mockKeychain,
	}

	// Need to create encryptor
	encryptor, err := NewEncryptor("chacha20-poly1305")
	require.NoError(t, err)
	store.encryptor = encryptor

	cred := &AuthCredential{
		AccessToken: "encrypted-token",
		Provider:    "test-provider",
		AuthMethod:  "token",
	}

	// Store credential
	err = store.SetCredential("test-provider", cred)
	require.NoError(t, err)

	// Retrieve credential
	retrieved, err := store.GetCredential("test-provider")
	require.NoError(t, err)
	assert.Equal(t, cred.AccessToken, retrieved.AccessToken)
}

func TestSecureStoreDeleteAll(t *testing.T) {
	ResetSecureStore()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	store, err := NewSecureStore(SecureStoreConfig{
		Enabled:     false,
		UseKeychain: false,
	})
	require.NoError(t, err)

	// Store multiple credentials
	for i := 0; i < 3; i++ {
		cred := &AuthCredential{
			AccessToken: "token-" + string(rune('a'+i)),
			Provider:    "provider-" + string(rune('a'+i)),
			AuthMethod:  "token",
		}
		err := store.SetCredential("provider-"+string(rune('a'+i)), cred)
		require.NoError(t, err)
	}

	// Delete all
	err = store.DeleteAllCredentials()
	require.NoError(t, err)

	// Verify all deleted
	providers, err := store.ListProviders()
	require.NoError(t, err)
	assert.Empty(t, providers)
}

func TestCredentialExpiry(t *testing.T) {
	// Test expired credential
	expiredCred := &AuthCredential{
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	assert.True(t, expiredCred.IsExpired())
	assert.True(t, expiredCred.NeedsRefresh())

	// Test valid credential
	validCred := &AuthCredential{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.False(t, validCred.IsExpired())
	assert.False(t, validCred.NeedsRefresh())

	// Test credential expiring soon
	expiringSoon := &AuthCredential{
		ExpiresAt: time.Now().Add(2 * time.Minute),
	}
	assert.False(t, expiringSoon.IsExpired())
	assert.True(t, expiringSoon.NeedsRefresh())

	// Test credential with no expiry
	noExpiry := &AuthCredential{
		ExpiresAt: time.Time{},
	}
	assert.False(t, noExpiry.IsExpired())
	assert.False(t, noExpiry.NeedsRefresh())
}
