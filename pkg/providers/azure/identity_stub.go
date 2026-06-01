//go:build !azidentity

// Stub for the Entra ID auth path when built without the azidentity tag.
// Mirrors the exported surface of identity.go so callers compile cleanly
// in the default build.

package azure

import "fmt"

const azidentityBuildHint = "azure identity auth not available: build with -tags azidentity to enable Entra ID auth, or set api_key"

// NewProviderWithIdentity returns an error in the default build.
func NewProviderWithIdentity(apiBase, proxy, userAgent string, opts ...Option) (*Provider, error) {
	return nil, fmt.Errorf("%s", azidentityBuildHint)
}

// NewProviderWithIdentityAndTimeout returns an error in the default build.
func NewProviderWithIdentityAndTimeout(
	apiBase, proxy, userAgent string,
	requestTimeoutSeconds int,
) (*Provider, error) {
	return nil, fmt.Errorf("%s", azidentityBuildHint)
}
