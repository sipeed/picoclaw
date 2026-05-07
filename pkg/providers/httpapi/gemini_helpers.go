package httpapi

import "strings"

func extractPartThoughtSignature(thoughtSignature string, thoughtSignatureSnake string) string {
	if thoughtSignature != "" {
		return thoughtSignature
	}
	if thoughtSignatureSnake != "" {
		return thoughtSignatureSnake
	}
	return ""
}

func extractProtocol(model string) (protocol, modelID string) {
	model = strings.TrimSpace(model)
	protocol, modelID, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return protocol, modelID
}
