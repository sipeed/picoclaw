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

type sandboxContextKey struct{}

// WithSandbox returns a derived context carrying the current sandbox instance.
func WithSandbox(ctx context.Context, sb Sandbox) context.Context {
	return context.WithValue(ctx, sandboxContextKey{}, sb)
}

// SandboxFromContext returns the sandbox instance attached by WithSandbox.
func SandboxFromContext(ctx context.Context) Sandbox {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(sandboxContextKey{}).(Sandbox)
	return v
}
