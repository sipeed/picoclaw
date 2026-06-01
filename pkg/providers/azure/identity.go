//go:build azidentity

// Entra ID (DefaultAzureCredential) auth adapter.
// Built only when -tags azidentity is supplied; otherwise identity_stub.go
// satisfies the same exported API with a friendly error.

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// azureOpenAIScope is the OAuth scope for Azure OpenAI (Cognitive Services).
// Service-wide scope, so it covers all regions including sovereign clouds.
const azureOpenAIScope = "https://cognitiveservices.azure.com/.default"

// NewProviderWithIdentity creates an Azure OpenAI provider authenticated via
// the DefaultAzureCredential chain (env vars, workload identity, managed
// identity, Azure CLI, ...). Construction itself only fails if the credential
// chain cannot be built; misconfigured environments surface their error on
// the first Chat call when GetToken is invoked.
func NewProviderWithIdentity(apiBase, proxy, userAgent string, opts ...Option) (*Provider, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating azure default credential: %w", err)
	}

	ts := func(ctx context.Context) (string, error) {
		tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{azureOpenAIScope},
		})
		if err != nil {
			return "", fmt.Errorf("acquiring azure access token: %w", err)
		}
		return tok.Token, nil
	}

	return NewProviderWithTokenSource(apiBase, proxy, userAgent, ts, opts...), nil
}

// NewProviderWithIdentityAndTimeout mirrors NewProviderWithTimeout for the
// identity auth path.
func NewProviderWithIdentityAndTimeout(
	apiBase, proxy, userAgent string,
	requestTimeoutSeconds int,
) (*Provider, error) {
	return NewProviderWithIdentity(
		apiBase, proxy, userAgent,
		WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
	)
}
