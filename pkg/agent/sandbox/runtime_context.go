package sandbox

import "context"

type sessionContextKey struct{}

// WithSessionKey returns a derived context carrying the current routing session key.
func WithSessionKey(ctx context.Context, sessionKey string) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, sessionKey)
}

// SessionKeyFromContext returns the session key attached by WithSessionKey.
func SessionKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(sessionContextKey{}).(string)
	return v
}
