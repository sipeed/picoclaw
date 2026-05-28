package providers

import (
	oauthprovider "github.com/sipeed/picoclaw/pkg/providers/oauth"
)

type (
	AntigravityProvider  = oauthprovider.AntigravityProvider
	AntigravityModelInfo = oauthprovider.AntigravityModelInfo
	ClaudeProvider       = oauthprovider.ClaudeProvider
	CodexProvider        = oauthprovider.CodexProvider
)

func NewAntigravityProvider() *AntigravityProvider {
	return oauthprovider.NewAntigravityProvider()
}

func NewClaudeProvider(token string) *ClaudeProvider {
	return oauthprovider.NewClaudeProvider(token)
}

func NewClaudeProviderWithBaseURL(token, apiBase string) *ClaudeProvider {
	return oauthprovider.NewClaudeProviderWithBaseURL(token, apiBase)
}

func NewClaudeProviderWithTokenSource(token string, tokenSource func() (string, error)) *ClaudeProvider {
	return oauthprovider.NewClaudeProviderWithTokenSource(token, tokenSource)
}

func NewClaudeProviderWithTokenSourceAndBaseURL(
	token string, tokenSource func() (string, error), apiBase string,
) *ClaudeProvider {
	return oauthprovider.NewClaudeProviderWithTokenSourceAndBaseURL(token, tokenSource, apiBase)
}

func NewCodexProvider(token, accountID string) *CodexProvider {
	return oauthprovider.NewCodexProvider(token, accountID)
}

func NewCodexProviderWithTokenSource(
	token, accountID string, tokenSource func() (string, string, error),
) *CodexProvider {
	return oauthprovider.NewCodexProviderWithTokenSource(token, accountID, tokenSource)
}

func FetchAntigravityProjectID(accessToken string) (string, error) {
	return oauthprovider.FetchAntigravityProjectID(accessToken)
}

func FetchAntigravityModels(accessToken, projectID string) ([]AntigravityModelInfo, error) {
	return oauthprovider.FetchAntigravityModels(accessToken, projectID)
}

func createClaudeTokenSource() func() (string, error) {
	return oauthprovider.CreateClaudeTokenSource(getCredential)
}

func createCodexTokenSource() func() (string, string, error) {
	return oauthprovider.CreateCodexTokenSource()
}
