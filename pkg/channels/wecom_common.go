// PicoClaw - Ultra-lightweight personal AI agent
// WeCom common utilities for both WeCom Bot and WeCom App

package channels

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

// WeComVerifySignature verifies the message signature for WeCom
// This is a common function used by both WeCom Bot and WeCom App
func WeComVerifySignature(token, msgSignature, timestamp, nonce, msgEncrypt string) bool {
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

// WeComDecryptMessage decrypts the encrypted message using AES
// This is a common function used by both WeCom Bot and WeCom App
func WeComDecryptMessage(encryptedMsg, encodingAESKey string) (string, error) {
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

	mode := cipher.NewCBCDecrypter(block, aesKey[:aes.BlockSize])
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// Remove PKCS7 padding
	plainText, err = pkcs7UnpadWeCom(plainText)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	// Parse message structure
	// Format: random(16) + msg_len(4) + msg + corp_id
	if len(plainText) < 20 {
		return "", fmt.Errorf("decrypted message too short")
	}

	msgLen := binary.BigEndian.Uint32(plainText[16:20])
	if int(msgLen) > len(plainText)-20 {
		return "", fmt.Errorf("invalid message length")
	}

	msg := plainText[20 : 20+msgLen]

	return string(msg), nil
}

// pkcs7UnpadWeCom removes PKCS7 padding with validation
func pkcs7UnpadWeCom(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize {
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
