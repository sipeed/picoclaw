package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/netbind"
	"github.com/sipeed/picoclaw/web/backend/middleware"
)

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

func TestShouldEnableLocalAutoLogin(t *testing.T) {
	tests := []struct {
		name       string
		noBrowser  bool
		probeHost  string
		wantEnable bool
	}{
		{name: "loopback localhost", probeHost: "localhost", wantEnable: true},
		{name: "loopback ipv4", probeHost: "127.0.0.1", wantEnable: true},
		{name: "loopback ipv6", probeHost: "::1", wantEnable: true},
		{name: "browser disabled", noBrowser: true, probeHost: "localhost", wantEnable: false},
		{name: "non-loopback host", probeHost: "192.168.1.50", wantEnable: false},
		{name: "non-loopback hostname", probeHost: "example.com", wantEnable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldEnableLocalAutoLogin(tt.noBrowser, tt.probeHost); got != tt.wantEnable {
				t.Fatalf(
					"shouldEnableLocalAutoLogin(%t, %q) = %t, want %t",
					tt.noBrowser,
					tt.probeHost,
					got,
					tt.wantEnable,
				)
			}
		})
	}
}

func TestLauncherBrowserLaunchSuffix(t *testing.T) {
	autoLogin, err := middleware.NewLauncherDashboardLocalAutoLogin(time.Minute)
	if err != nil {
		t.Fatalf("NewLauncherDashboardLocalAutoLogin() error = %v", err)
	}

	if got := launcherBrowserLaunchSuffix(true, autoLogin); got != middleware.LauncherDashboardSetupPath {
		t.Fatalf("setup suffix = %q", got)
	}
	if got := launcherBrowserLaunchSuffix(false, autoLogin); !strings.HasPrefix(got, "/launcher-auto-login?nonce=") {
		t.Fatalf("auto-login suffix = %q", got)
	}
	if got := launcherBrowserLaunchSuffix(false, nil); got != "" {
		t.Fatalf("empty suffix = %q, want empty", got)
	}
}

func TestResolveLauncherHostInput(t *testing.T) {
	tests := []struct {
		name         string
		flagHost     string
		explicitFlag bool
		envHost      string
		wantHost     string
		wantActive   bool
		wantErr      bool
	}{
		{
			name:         "flag host wins",
			flagHost:     "127.0.0.1",
			explicitFlag: true,
			envHost:      "::",
			wantHost:     "127.0.0.1",
			wantActive:   true,
		},
		{name: "env host used when flag absent", envHost: "127.0.0.1,::1", wantHost: "127.0.0.1,::1", wantActive: true},
		{name: "blank env ignored", envHost: "   ", wantHost: "", wantActive: false},
		{name: "invalid flag rejected", flagHost: "127.0.0.1, ", explicitFlag: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotActive, err := resolveLauncherHostInput(tt.flagHost, tt.explicitFlag, tt.envHost)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveLauncherHostInput() err = %v, wantErr %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotHost != tt.wantHost {
				t.Fatalf("resolveLauncherHostInput() host = %q, want %q", gotHost, tt.wantHost)
			}
			if gotActive != tt.wantActive {
				t.Fatalf("resolveLauncherHostInput() active = %t, want %t", gotActive, tt.wantActive)
			}
		})
	}
}

func TestLauncherConsoleHosts(t *testing.T) {
	t.Run("default loopback shows localhost only", func(t *testing.T) {
		hosts := launcherConsoleHostsWithLocalAddrs(
			"",
			false,
			[]string{"192.168.1.2", "10.0.0.8"},
			[]string{"2001:db8::1", "2001:db8::2"},
		)
		want := []string{"localhost"}
		if strings.Join(hosts, ",") != strings.Join(want, ",") {
			t.Fatalf("hosts = %#v, want %#v", hosts, want)
		}
	})

	t.Run("explicit loopback hosts collapse to localhost", func(t *testing.T) {
		tests := []struct {
			name      string
			hostInput string
		}{
			{name: "ipv6 loopback", hostInput: "::1"},
			{name: "ipv4 loopback", hostInput: "127.0.0.1"},
			{name: "localhost", hostInput: "localhost"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hosts := launcherConsoleHostsWithLocalAddrs(
					tt.hostInput,
					false,
					[]string{"192.168.1.2", "10.0.0.8"},
					[]string{"2001:db8::1", "2001:db8::2"},
				)
				want := []string{"localhost"}
				if strings.Join(hosts, ",") != strings.Join(want, ",") {
					t.Fatalf("hosts = %#v, want %#v", hosts, want)
				}
			})
		}
	})

	t.Run("public wildcard shows localhost then ipv6 and ipv4", func(t *testing.T) {
		hosts := launcherConsoleHostsWithLocalAddrs(
			"",
			true,
			[]string{"192.168.1.2", "10.0.0.8"},
			[]string{"2001:db8::1", "2001:db8::2"},
		)
		want := []string{"localhost", "2001:db8::1", "2001:db8::2", "192.168.1.2", "10.0.0.8"}
		if strings.Join(hosts, ",") != strings.Join(want, ",") {
			t.Fatalf("hosts = %#v, want %#v", hosts, want)
		}
	})

	t.Run("explicit ipv6 any shows localhost then ipv6 variants", func(t *testing.T) {
		hosts := launcherConsoleHostsWithLocalAddrs(
			"::",
			false,
			[]string{"192.168.1.2", "10.0.0.8"},
			[]string{"2001:db8::1", "2001:db8::2"},
		)
		want := []string{"localhost", "2001:db8::1", "2001:db8::2"}
		if strings.Join(hosts, ",") != strings.Join(want, ",") {
			t.Fatalf("hosts = %#v, want %#v", hosts, want)
		}

		for _, host := range hosts {
			if host == "::1" || host == "127.0.0.1" || strings.HasPrefix(strings.ToLower(host), "fe80:") {
				t.Fatalf("hosts = %#v, loopback IPs must not be displayed", hosts)
			}
		}
	})

	t.Run("explicit ipv4 any shows localhost then lan ipv4", func(t *testing.T) {
		hosts := launcherConsoleHostsWithLocalAddrs(
			"0.0.0.0",
			false,
			[]string{"192.168.1.2", "10.0.0.8"},
			[]string{"2001:db8::1", "2001:db8::2"},
		)
		want := []string{"localhost", "192.168.1.2", "10.0.0.8"}
		if strings.Join(hosts, ",") != strings.Join(want, ",") {
			t.Fatalf("hosts = %#v, want %#v", hosts, want)
		}
	})

	t.Run("explicit wildcard star shows localhost first", func(t *testing.T) {
		hosts := launcherConsoleHostsWithLocalAddrs(
			"*",
			false,
			[]string{"192.168.1.2", "10.0.0.8"},
			[]string{"2001:db8::1", "2001:db8::2"},
		)
		want := []string{"localhost", "2001:db8::1", "2001:db8::2", "192.168.1.2", "10.0.0.8"}
		if strings.Join(hosts, ",") != strings.Join(want, ",") {
			t.Fatalf("hosts = %#v, want %#v", hosts, want)
		}
	})

	t.Run("explicit multi-address binding without local tokens hides localhost", func(t *testing.T) {
		hosts := launcherConsoleHostsWithLocalAddrs(
			"192.168.1.2,10.0.0.8,2001:db8::1,2001:db8::2,fe80::1",
			false,
			[]string{"192.168.1.2", "10.0.0.8"},
			[]string{"2001:db8::1", "2001:db8::2"},
		)
		want := []string{"192.168.1.2", "10.0.0.8", "2001:db8::1", "2001:db8::2"}
		if strings.Join(hosts, ",") != strings.Join(want, ",") {
			t.Fatalf("hosts = %#v, want %#v", hosts, want)
		}
	})
}

func TestWildcardAdvertiseIP(t *testing.T) {
	tests := []struct {
		name      string
		bindHosts []string
		ipv4      string
		ipv6      string
		want      string
	}{
		{
			name:      "ipv4 wildcard uses ipv4",
			bindHosts: []string{"0.0.0.0"},
			ipv4:      "192.168.1.2",
			ipv6:      "2001:db8::1",
			want:      "192.168.1.2",
		},
		{
			name:      "dual wildcard prefers ipv6",
			bindHosts: []string{"0.0.0.0", "::"},
			ipv4:      "192.168.1.2",
			ipv6:      "2001:db8::1",
			want:      "2001:db8::1",
		},
		{
			name:      "ipv6 wildcard uses ipv6",
			bindHosts: []string{"::"},
			ipv4:      "192.168.1.2",
			ipv6:      "2001:db8::1",
			want:      "2001:db8::1",
		},
		{
			name:      "dual wildcard falls back to ipv4 when ipv6 missing",
			bindHosts: []string{"0.0.0.0", "::"},
			ipv4:      "192.168.1.2",
			ipv6:      "",
			want:      "192.168.1.2",
		},
		{
			name:      "ipv6 wildcard without ipv6 does not advertise ipv4",
			bindHosts: []string{"::"},
			ipv4:      "192.168.1.2",
			ipv6:      "",
			want:      "",
		},
		{
			name:      "non wildcard does not advertise",
			bindHosts: []string{"127.0.0.1"},
			ipv4:      "192.168.1.2",
			ipv6:      "2001:db8::1",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wildcardAdvertiseIP(tt.bindHosts, tt.ipv4, tt.ipv6); got != tt.want {
				t.Fatalf("wildcardAdvertiseIP(%#v, %q, %q) = %q, want %q", tt.bindHosts, tt.ipv4, tt.ipv6, got, tt.want)
			}
		})
	}
}

func TestOpenLauncherListeners_HonorsIPv6OnlyHost(t *testing.T) {
	hasIPv4, hasIPv6 := netbind.DetectIPFamilies()
	if !hasIPv6 {
		t.Skip("IPv6 is unavailable in this environment")
	}

	result, err := openLauncherListeners("::", false, "0")
	if err != nil {
		t.Fatalf("openLauncherListeners() error = %v", err)
	}
	startLauncherTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	requireLauncherHTTPReachable(t, "::1", port)
	if hasIPv4 {
		requireLauncherHTTPUnreachable(t, "127.0.0.1", port)
	}
}

func TestOpenLauncherListeners_SupportsExplicitMultiHost(t *testing.T) {
	hasIPv4, hasIPv6 := netbind.DetectIPFamilies()
	if !hasIPv4 || !hasIPv6 {
		t.Skip("dual-stack loopback is unavailable in this environment")
	}

	result, err := openLauncherListeners("127.0.0.1,::1", false, "0")
	if err != nil {
		t.Fatalf("openLauncherListeners() error = %v", err)
	}
	startLauncherTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	requireLauncherHTTPReachable(t, "127.0.0.1", port)
	requireLauncherHTTPReachable(t, "::1", port)
}

func startLauncherTestHTTPServer(t *testing.T, listeners []net.Listener) {
	t.Helper()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ok")
		}),
	}

	errCh := make(chan error, len(listeners))
	for _, listener := range listeners {
		ln := listener
		go func() {
			errCh <- server.Serve(ln)
		}()
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		for range listeners {
			err := <-errCh
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Fatalf("server.Serve() error = %v", err)
			}
		}
	})
}

func requireLauncherHTTPReachable(t *testing.T, host string, port int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := launcherHTTPGet(host, port)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %s:%d to be reachable: %v", host, port, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func requireLauncherHTTPUnreachable(t *testing.T, host string, port int) {
	t.Helper()
	if err := launcherHTTPGet(host, port); err == nil {
		t.Fatalf("expected %s:%d to be unreachable", host, port)
	}
}

func launcherHTTPGet(host string, port int) error {
	client := &http.Client{
		Timeout: 300 * time.Millisecond,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	resp, err := client.Get("http://" + net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	n, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v", value, err)
	}
	return n
}
