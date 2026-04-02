package main

import "testing"

func TestShouldEnableLauncherFileLogging(t *testing.T) {
	tests := []struct {
		name          string
		enableConsole bool
		debug         bool
		want          bool
	}{
		{name: "gui mode", enableConsole: false, debug: false, want: true},
		{name: "console mode", enableConsole: true, debug: false, want: false},
		{name: "debug gui mode", enableConsole: false, debug: true, want: true},
		{name: "debug console mode", enableConsole: true, debug: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldEnableLauncherFileLogging(tt.enableConsole, tt.debug); got != tt.want {
				t.Fatalf(
					"shouldEnableLauncherFileLogging(%t, %t) = %t, want %t",
					tt.enableConsole,
					tt.debug,
					got,
					tt.want,
				)
			}
		})
	}
}
