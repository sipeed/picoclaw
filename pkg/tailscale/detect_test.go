package tailscale

import (
	"testing"
)

func TestParseHostname(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    string
		wantErr bool
	}{
		{
			name: "valid hostname with trailing dot",
			json: `{"Self":{"DNSName":"mybox.tail1234.ts.net."}}`,
			want: "mybox.tail1234.ts.net",
		},
		{
			name: "valid hostname without trailing dot",
			json: `{"Self":{"DNSName":"mybox.tail1234.ts.net"}}`,
			want: "mybox.tail1234.ts.net",
		},
		{
			name:    "empty DNSName",
			json:    `{"Self":{"DNSName":""}}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			json:    `not json`,
			wantErr: true,
		},
		{
			name:    "missing Self field",
			json:    `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHostname([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseHostname() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseHostname() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseHostname() = %q, want %q", got, tt.want)
			}
		})
	}
}
