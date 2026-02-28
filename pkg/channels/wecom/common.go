package wecom

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
)

// blockSize is the PKCS7 block size used by WeCom (32)
const blockSize = 32

// verifySignature verifies the message signature for WeCom
// This is a common function used by both WeCom Bot and WeCom App
func verifySignature(token, msgSignature, timestamp, nonce, msgEncrypt string) bool {
	if token == "" {
		return true // Skip verification if token is not set
	}

	// Sort parameters
	params := []string{token, timestamp, nonce, msgEncrypt}
	sort.Strings(params)

	// Concatenate
	str := strings.Join(params, "")

	// SHA1 hash
	hash := sha1.Sum([]byte(str))
	expectedSignature := fmt.Sprintf("%x", hash)

	return expectedSignature == msgSignature
}

// decryptMessage decrypts the encrypted message using AES
// For AIBOT, receiveid should be the aibotid; for other apps, it should be corp_id
func decryptMessage(encryptedMsg, encodingAESKey string) (string, error) {
	return decryptMessageWithVerify(encryptedMsg, encodingAESKey, "")
}

// decryptMessageWithVerify decrypts the encrypted message and optionally verifies receiveid
// receiveid: for AIBOT use aibotid, for WeCom App use corp_id. If empty, skip verification.
func decryptMessageWithVerify(encryptedMsg, encodingAESKey, receiveid string) (string, error) {
	if encodingAESKey == "" {
		// No encryption, return as is (base64 decode)
		decoded, err := base64.StdEncoding.DecodeString(encryptedMsg)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	// Decode AES key (base64)
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return "", fmt.Errorf("failed to decode AES key: %w", err)
	}

	// Decode encrypted message
	cipherText, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		return "", fmt.Errorf("failed to decode message: %w", err)
	}

	// AES decrypt
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// IV is the first 16 bytes of AESKey
	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// Remove PKCS7 padding
	plainText, err = pkcs7Unpad(plainText)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	// Parse message structure
	// Format: random(16) + msg_len(4) + msg + receiveid
	if len(plainText) < 20 {
		return "", fmt.Errorf("decrypted message too short")
	}

	msgLen := binary.BigEndian.Uint32(plainText[16:20])
	if int(msgLen) > len(plainText)-20 {
		return "", fmt.Errorf("invalid message length")
	}

	msg := plainText[20 : 20+msgLen]

	// Verify receiveid if provided
	if receiveid != "" && len(plainText) > 20+int(msgLen) {
		actualReceiveID := string(plainText[20+msgLen:])
		if actualReceiveID != receiveid {
			return "", fmt.Errorf("receiveid mismatch: expected %s, got %s", receiveid, actualReceiveID)
		}
	}

	return string(msg), nil
}

// pkcs7Unpad removes PKCS7 padding with validation
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	padding := int(data[len(data)-1])
	// WeCom uses 32-byte block size for PKCS7 padding
	if padding == 0 || padding > blockSize {
		return nil, fmt.Errorf("invalid padding size: %d", padding)
	}
	if padding > len(data) {
		return nil, fmt.Errorf("padding size larger than data")
	}
	// Verify all padding bytes
	for i := 0; i < padding; i++ {
		if data[len(data)-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding byte at position %d", i)
		}
	}
	return data[:len(data)-padding], nil
}
