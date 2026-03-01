package wecom

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// blockSize is the PKCS7 block size used by WeCom (32)
const blockSize = 32

// computeSignature computes the WeCom message signature from the given parameters.
// It sorts [token, timestamp, nonce, encrypt], concatenates them and returns the SHA1 hex digest.
func computeSignature(token, timestamp, nonce, encrypt string) string {
	params := []string{token, timestamp, nonce, encrypt}
	sort.Strings(params)
	str := strings.Join(params, "")
	hash := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// verifySignature verifies the message signature for WeCom
// This is a common function used by both WeCom Bot and WeCom App
func verifySignature(token, msgSignature, timestamp, nonce, msgEncrypt string) bool {
	if token == "" {
		return true // Skip verification if token is not set
	}
	return computeSignature(token, timestamp, nonce, msgEncrypt) == msgSignature
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

	aesKey, err := decodeWeComAESKey(encodingAESKey)
	if err != nil {
		return "", err
	}

	cipherText, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		return "", fmt.Errorf("failed to decode message: %w", err)
	}

	plainText, err := decryptAESCBC(aesKey, cipherText)
	if err != nil {
		return "", err
	}

	return unpackWeComFrame(plainText, receiveid)
}

// decodeWeComAESKey base64-decodes the 43-character EncodingAESKey (trailing "=" is
// appended automatically) and validates that the result is exactly 32 bytes.
// It is the single place that handles this repeated pattern in both encrypt and decrypt paths.
func decodeWeComAESKey(encodingAESKey string) ([]byte, error) {
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("invalid AES key length: %d", len(aesKey))
	}
	return aesKey, nil
}

// encryptAESCBC encrypts plaintext using AES-CBC with the given key, mirroring
// decryptAESCBC. IV = aesKey[:aes.BlockSize]. The caller must PKCS7-pad the
// plaintext to a multiple of aes.BlockSize before calling.
func encryptAESCBC(aesKey, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	iv := aesKey[:aes.BlockSize]
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	return ciphertext, nil
}

// packWeComFrame builds the WeCom wire format:
//
//	random(16 ASCII digits) + msg_len(4, big-endian) + msg + receiveid
func packWeComFrame(msg, receiveid string) ([]byte, error) {
	randomBytes := make([]byte, 16)
	for i := range 16 {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return nil, fmt.Errorf("failed to generate random: %w", err)
		}
		randomBytes[i] = byte('0' + n.Int64())
	}
	msgBytes := []byte(msg)
	msgLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBytes, uint32(len(msgBytes)))
	var buf bytes.Buffer
	buf.Write(randomBytes)
	buf.Write(msgLenBytes)
	buf.Write(msgBytes)
	buf.WriteString(receiveid)
	return buf.Bytes(), nil
}

// unpackWeComFrame parses the WeCom wire format produced by packWeComFrame.
// If receiveid is non-empty it verifies the frame's trailing receiveid field.
func unpackWeComFrame(data []byte, receiveid string) (string, error) {
	if len(data) < 20 {
		return "", fmt.Errorf("decrypted frame too short: %d bytes", len(data))
	}
	msgLen := binary.BigEndian.Uint32(data[16:20])
	if int(msgLen) > len(data)-20 {
		return "", fmt.Errorf("invalid message length: %d", msgLen)
	}
	msg := data[20 : 20+msgLen]
	if receiveid != "" && len(data) > 20+int(msgLen) {
		actualReceiveID := string(data[20+msgLen:])
		if actualReceiveID != receiveid {
			return "", fmt.Errorf("receiveid mismatch: expected %s, got %s", receiveid, actualReceiveID)
		}
	}
	return string(msg), nil
}

// decryptAESCBC decrypts ciphertext using AES-CBC with the given key.
// IV = aesKey[:aes.BlockSize]. PKCS7 padding is stripped from the returned plaintext.
func decryptAESCBC(aesKey, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is empty")
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size", len(ciphertext))
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	iv := aesKey[:aes.BlockSize]
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to unpad: %w", err)
	}
	return plaintext, nil
}

// pkcs7Pad adds PKCS7 padding
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	if padding == 0 {
		padding = blockSize
	}
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
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
	for i := range padding {
		if data[len(data)-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding byte at position %d", i)
		}
	}
	return data[:len(data)-padding], nil
}
