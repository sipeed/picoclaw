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

	"github.com/sipeed/picoclaw/pkg/logger"
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
// For AIBOT, receiveid should be the aibotid; for other apps, it should be corp_id
func WeComDecryptMessage(encryptedMsg, encodingAESKey string) (string, error) {
	return WeComDecryptMessageWithVerify(encryptedMsg, encodingAESKey, "")
}

// WeComDecryptMessageWithVerify decrypts the encrypted message and optionally verifies receiveid
// receiveid: for AIBOT use aibotid, for WeCom App use corp_id. If empty, skip verification.
func WeComDecryptMessageWithVerify(encryptedMsg, encodingAESKey, receiveid string) (string, error) {
	logger.DebugCF("wecom_common", "Starting decryption", map[string]interface{}{
		"encodingAESKey_len": len(encodingAESKey),
		"receiveid":          receiveid,
		"encryptedMsg_len":   len(encryptedMsg),
	})

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
		logger.ErrorCF("wecom_common", "Failed to decode AES key", map[string]interface{}{
			"error": err.Error(),
			"key":   encodingAESKey,
		})
		return "", fmt.Errorf("failed to decode AES key: %w", err)
	}
	logger.DebugCF("wecom_common", "AES key decoded", map[string]interface{}{
		"key_len": len(aesKey),
	})

	// Decode encrypted message
	cipherText, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		logger.ErrorCF("wecom_common", "Failed to decode message", map[string]interface{}{
			"error": err.Error(),
		})
		return "", fmt.Errorf("failed to decode message: %w", err)
	}
	logger.DebugCF("wecom_common", "Message decoded", map[string]interface{}{
		"cipher_len": len(cipherText),
	})

	// AES decrypt
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short: %d < %d", len(cipherText), aes.BlockSize)
	}

	// IV is the first 16 bytes of AESKey
	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// Remove PKCS7 padding
	unpaddedText, err := pkcs7UnpadWeCom(plainText)
	if err != nil {
		lastByte := -1
		if len(plainText) > 0 {
			lastByte = int(plainText[len(plainText)-1])
		}
		logger.ErrorCF("wecom_common", "PKCS7 unpad failed", map[string]interface{}{
			"error":     err.Error(),
			"plain_len": len(plainText),
			"last_byte": lastByte,
		})
		return "", fmt.Errorf("failed to unpad: %w", err)
	}
	plainText = unpaddedText

	// Parse message structure
	// Format: random(16) + msg_len(4) + msg + receiveid
	if len(plainText) < 20 {
		return "", fmt.Errorf("decrypted message too short")
	}

	msgLen := binary.BigEndian.Uint32(plainText[16:20])
	logger.DebugCF("wecom_common", "Message structure parsed", map[string]interface{}{
		"msg_len":        msgLen,
		"plain_len":      len(plainText),
		"total_expected": 20 + int(msgLen),
	})

	if int(msgLen) > len(plainText)-20 {
		return "", fmt.Errorf("invalid message length: %d > %d", msgLen, len(plainText)-20)
	}

	msg := plainText[20 : 20+msgLen]

	// Verify receiveid if provided
	if receiveid != "" && len(plainText) > 20+int(msgLen) {
		actualReceiveID := string(plainText[20+msgLen:])
		logger.DebugCF("wecom_common", "ReceiveID verification", map[string]interface{}{
			"expected": receiveid,
			"actual":   actualReceiveID,
		})
		if actualReceiveID != receiveid {
			return "", fmt.Errorf("receiveid mismatch: expected %s, got %s", receiveid, actualReceiveID)
		}
	}

	logger.DebugCF("wecom_common", "Decryption successful", map[string]interface{}{
		"msg_len": len(msg),
	})
	return string(msg), nil
}

// pkcs7UnpadWeCom removes PKCS7 padding with validation
// WeCom uses block size of 32 (not standard AES block size of 16)
const wecomBlockSize = 32

func pkcs7UnpadWeCom(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	padding := int(data[len(data)-1])
	// WeCom uses 32-byte block size for PKCS7 padding
	if padding == 0 || padding > wecomBlockSize {
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
