package auth

import "fmt"

// GetOpenAIToken returns the current OpenAI credential, refreshing OAuth
// credentials when they are close to expiry. The account ID is returned for
// Codex backend calls that require the Chatgpt-Account-Id header.
func GetOpenAIToken() (accessToken, accountID string, err error) {
	cred, err := GetCredential("openai")
	if err != nil {
		return "", "", fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return "", "", fmt.Errorf("no credentials for openai. Run: picoclaw auth login --provider openai")
	}

	if cred.AuthMethod == "oauth" && cred.NeedsRefresh() && cred.RefreshToken != "" {
		refreshed, err := RefreshAccessToken(cred, OpenAIOAuthConfig())
		if err != nil {
			return "", "", fmt.Errorf("refreshing token: %w", err)
		}
		if refreshed.AccountID == "" {
			refreshed.AccountID = cred.AccountID
		}
		if err := SetCredential("openai", refreshed); err != nil {
			return "", "", fmt.Errorf("saving refreshed token: %w", err)
		}
		return refreshed.AccessToken, refreshed.AccountID, nil
	}

	return cred.AccessToken, cred.AccountID, nil
}
