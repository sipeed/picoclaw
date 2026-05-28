package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGatewayCommand(t *testing.T) {
	cmd := NewGatewayCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "gateway", cmd.Use)
	assert.Equal(t, "Start picoclaw gateway", cmd.Short)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("g"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.False(t, cmd.HasSubCommands())

	assert.True(t, cmd.HasFlags())
	assert.NotNil(t, cmd.Flags().Lookup("debug"))
	assert.NotNil(t, cmd.Flags().Lookup("allow-empty"))
	assert.NotNil(t, cmd.Flags().Lookup("host"))
}

func TestResolveGatewayHostOverride(t *testing.T) {
	tests := []struct {
		name     string
		explicit bool
		host     string
		wantHost string
		wantErr  bool
	}{
		{name: "implicit empty host is allowed", explicit: false, host: "", wantHost: "", wantErr: false},
		{name: "explicit empty host rejected", explicit: true, host: "   ", wantHost: "", wantErr: true},
		{name: "explicit localhost kept", explicit: true, host: " localhost ", wantHost: "localhost", wantErr: false},
		{
			name:     "explicit multi host normalized",
			explicit: true,
			host:     " [::1] , 127.0.0.1 ",
			wantHost: "::1,127.0.0.1",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveGatewayHostOverride(tt.explicit, tt.host)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveGatewayHostOverride() err = %v, wantErr %t", err, tt.wantErr)
			}
			if got != tt.wantHost {
				t.Fatalf("resolveGatewayHostOverride() host = %q, want %q", got, tt.wantHost)
			}
		})
	}
}
