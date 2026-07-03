package discord

import (
	"reflect"
	"testing"
)

func TestHasAllowedRole(t *testing.T) {
	tests := []struct {
		name        string
		memberRoles []string
		allowRoles  []string
		want        bool
	}{
		{
			name:        "matching role returns true",
			memberRoles: []string{"111", "222"},
			allowRoles:  []string{"222"},
			want:        true,
		},
		{
			name:        "non-matching role returns false",
			memberRoles: []string{"111", "333"},
			allowRoles:  []string{"222"},
			want:        false,
		},
		{
			name:        "empty allow_roles returns true (no restriction)",
			memberRoles: []string{"111"},
			allowRoles:  []string{},
			want:        true,
		},
		{
			name:        "nil member roles returns false when allow_roles set",
			memberRoles: nil,
			allowRoles:  []string{"222"},
			want:        false,
		},
		{
			name:        "multiple allow roles, one matches",
			memberRoles: []string{"aaa", "bbb"},
			allowRoles:  []string{"xxx", "bbb", "yyy"},
			want:        true,
		},
		{
			name:        "empty member roles, allow_roles set",
			memberRoles: []string{},
			allowRoles:  []string{"222"},
			want:        false,
		},
		{
			name:        "both empty returns true",
			memberRoles: []string{},
			allowRoles:  []string{},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAllowedRole(tt.memberRoles, tt.allowRoles)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("hasAllowedRole(%v, %v) = %v, want %v",
					tt.memberRoles, tt.allowRoles, got, tt.want)
			}
		})
	}
}
