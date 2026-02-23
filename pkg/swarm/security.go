// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package swarm

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// AuthProvider handles authentication for swarm nodes.
// Uses HMAC-based shared secret authentication.
type AuthProvider struct {
	sharedSecret []byte
	nodeID       string
}

// NewAuthProvider creates a new authentication provider.
func NewAuthProvider(nodeID, sharedSecret string) *AuthProvider {
	return &AuthProvider{
		sharedSecret: []byte(sharedSecret),
		nodeID:       nodeID,
	}
}

// SignMessage signs a message with HMAC-SHA256.
// The signature is base64 encoded for JSON transmission.
func (a *AuthProvider) SignMessage(msg any) (string, error) {
	if a.sharedSecret == nil {
		return "", ErrAuthenticationFailed
	}

	// Serialize message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Calculate HMAC
	h := hmac.New(sha256.New, a.sharedSecret)
	h.Write(data)
	signature := h.Sum(nil)

	// Return base64 encoded signature
	return base64.StdEncoding.EncodeToString(signature), nil
}

// VerifySignature verifies a message signature.
func (a *AuthProvider) VerifySignature(msg any, signature string) bool {
	if a.sharedSecret == nil {
		return false
	}

	// Calculate expected signature
	expected, err := a.SignMessage(msg)
	if err != nil {
		return false
	}

	// Compare signatures
	return hmac.Equal([]byte(expected), []byte(signature))
}

// GetNodeID returns the node ID for this auth provider.
func (a *AuthProvider) GetNodeID() string {
	return a.nodeID
}

// AuthToken represents an authentication token.
type AuthToken struct {
	NodeID    string `json:"node_id"`
	Signature string `json:"signature"`
	Timestamp int64  `json:"timestamp"`
}

// GenerateToken creates an auth token for the given node.
func (a *AuthProvider) GenerateToken() (*AuthToken, error) {
	token := &AuthToken{
		NodeID:    a.nodeID,
		Timestamp: time.Now().UnixNano(),
	}

	signature, err := a.SignMessage(token)
	if err != nil {
		return nil, err
	}

	token.Signature = signature
	return token, nil
}

// VerifyToken verifies an auth token.
func (a *AuthProvider) VerifyToken(token *AuthToken) bool {
	if token == nil {
		return false
	}

	// Check token age (reject tokens older than 1 minute)
	age := time.Since(time.Unix(0, token.Timestamp))
	if age > time.Minute {
		return false
	}

	return a.VerifySignature(token, token.Signature)
}

// AuthenticateNode verifies that a node is allowed to join.
func (a *AuthProvider) AuthenticateNode(nodeID, signature string, challengeData any) bool {
	if a.sharedSecret == nil {
		// No authentication configured - allow all
		return true
	}

	challenge := map[string]any{
		"node_id": nodeID,
		"data":    challengeData,
	}

	return a.VerifySignature(challenge, signature)
}
