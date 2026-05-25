package api

import (
	"strings"
	"testing"
)

func TestMaskAPIKeyValue(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "empty key",
			key:  "",
			want: "",
		},
		{
			name: "whitespace only",
			key:  "   ",
			want: "",
		},
		{
			name: "short key fully masked",
			key:  "abcd",
			want: "****",
		},
		{
			name: "length 8 boundary fully masked",
			key:  "12345678",
			want: "****",
		},
		{
			name: "length 9 boundary shows last 2",
			key:  "123456789",
			want: "123****89",
		},
		{
			name: "length 10 shows last 2",
			key:  "1234567890",
			want: "123****90",
		},
		{
			name: "length 12 boundary shows last 2",
			key:  "abcdefghijkl",
			want: "abc****kl",
		},
		{
			name: "length 13 boundary shows last 4",
			key:  "abcdefghijklm",
			want: "abc****jklm",
		},
		{
			name: "typical api key",
			key:  "sk-1234567890abcd",
			want: "sk-****abcd",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := maskAPIKeyValue(tc.key)
			if got != tc.want {
				t.Fatalf("maskAPIKeyValue(%q) = %q, want %q", tc.key, got, tc.want)
			}

			if tc.key != "" {
				displayed := strings.Replace(got, "****", "", 1)
				if len(strings.TrimSpace(tc.key)) <= 8 {
					if displayed != "" {
						t.Fatalf("maskAPIKeyValue(%q) displayed part = %q, want empty", tc.key, displayed)
					}
				} else {
					if len(displayed)*10 > len(strings.TrimSpace(tc.key))*6 {
						t.Fatalf(
							"maskAPIKeyValue(%q) displayed length = %d, want at most 60%% of %d",
							tc.key,
							len(displayed),
							len(strings.TrimSpace(tc.key)),
						)
					}
				}
			}
		})
	}
}
