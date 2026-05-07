package netbind

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestNormalizeHostInput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "single host", raw: "127.0.0.1", want: "127.0.0.1"},
		{name: "trim and dedupe", raw: " [::1] , ::1 , 127.0.0.1 ", want: "::1,127.0.0.1"},
		{name: "star preserved", raw: "*,127.0.0.1", want: "*,127.0.0.1"},
		{name: "reject empty", raw: "127.0.0.1, ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeHostInput(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeHostInput() err = %v, wantErr %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("NormalizeHostInput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPlan_DefaultAnyUsesLoopbackProbe(t *testing.T) {
	plan, err := BuildPlan("", DefaultAny)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.ProbeHost != ResolveAdaptiveLoopbackHost() {
		t.Fatalf("ProbeHost = %q, want %q", plan.ProbeHost, ResolveAdaptiveLoopbackHost())
	}
}

func TestOpenPlan_LocalhostSupportsLoopbackCommunication(t *testing.T) {
	hasIPv4, hasIPv6 := DetectIPFamilies()

	plan, err := BuildPlan("localhost", DefaultLoopback)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := OpenPlan(plan, "0")
	if err != nil {
		t.Fatalf("OpenPlan() error = %v", err)
	}
	startTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	if hasIPv6 {
		requireHTTPReachable(t, "::1", port)
	}
	if hasIPv4 {
		requireHTTPReachable(t, "127.0.0.1", port)
	}
}

func TestOpenPlan_DefaultAnySupportsDualStackLoopback(t *testing.T) {
	hasIPv4, hasIPv6 := DetectIPFamilies()

	plan, err := BuildPlan("", DefaultAny)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := OpenPlan(plan, "0")
	if err != nil {
		t.Fatalf("OpenPlan() error = %v", err)
	}
	startTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	if hasIPv6 {
		requireHTTPReachable(t, "::1", port)
	}
	if hasIPv4 {
		requireHTTPReachable(t, "127.0.0.1", port)
	}

	switch {
	case hasIPv4 && hasIPv6:
		if len(result.BindHosts) != 2 {
			t.Fatalf("len(BindHosts) = %d, want 2 (%#v)", len(result.BindHosts), result.BindHosts)
		}
	case hasIPv6 || hasIPv4:
		if len(result.BindHosts) != 1 {
			t.Fatalf("len(BindHosts) = %d, want 1 (%#v)", len(result.BindHosts), result.BindHosts)
		}
	}
}

func TestOpenPlan_ExplicitIPv6AnyIsIPv6Only(t *testing.T) {
	hasIPv4, hasIPv6 := DetectIPFamilies()
	if !hasIPv6 {
		t.Skip("IPv6 is unavailable in this environment")
	}

	plan, err := BuildPlan("::", DefaultLoopback)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := OpenPlan(plan, "0")
	if err != nil {
		t.Fatalf("OpenPlan() error = %v", err)
	}
	startTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	requireHTTPReachable(t, "::1", port)
	if hasIPv4 {
		requireHTTPUnreachable(t, "127.0.0.1", port)
	}
}

func TestOpenPlan_ExplicitIPv4AnyIsIPv4Only(t *testing.T) {
	hasIPv4, hasIPv6 := DetectIPFamilies()
	if !hasIPv4 {
		t.Skip("IPv4 is unavailable in this environment")
	}

	plan, err := BuildPlan("0.0.0.0", DefaultLoopback)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := OpenPlan(plan, "0")
	if err != nil {
		t.Fatalf("OpenPlan() error = %v", err)
	}
	startTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	requireHTTPReachable(t, "127.0.0.1", port)
	if hasIPv6 {
		requireHTTPUnreachable(t, "::1", port)
	}
}

func TestOpenPlan_MultiHostSupportsExplicitIPv4AndIPv6(t *testing.T) {
	hasIPv4, hasIPv6 := DetectIPFamilies()
	if !hasIPv4 || !hasIPv6 {
		t.Skip("dual-stack loopback is unavailable in this environment")
	}

	plan, err := BuildPlan("127.0.0.1,::1", DefaultLoopback)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := OpenPlan(plan, "0")
	if err != nil {
		t.Fatalf("OpenPlan() error = %v", err)
	}
	startTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	requireHTTPReachable(t, "127.0.0.1", port)
	requireHTTPReachable(t, "::1", port)
}

func TestOpenPlan_WildcardRulesKeepIPv4AndIPv6AnyHosts(t *testing.T) {
	hasIPv4, hasIPv6 := DetectIPFamilies()
	if !hasIPv4 || !hasIPv6 {
		t.Skip("dual-stack loopback is unavailable in this environment")
	}

	plan, err := BuildPlan("::,::1,0.0.0.0,127.0.0.1", DefaultLoopback)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	result, err := OpenPlan(plan, "0")
	if err != nil {
		t.Fatalf("OpenPlan() error = %v", err)
	}
	startTestHTTPServer(t, result.Listeners)
	port := mustAtoi(t, result.Port)

	requireHTTPReachable(t, "127.0.0.1", port)
	requireHTTPReachable(t, "::1", port)
	if len(result.BindHosts) != 2 {
		t.Fatalf("len(BindHosts) = %d, want 2 (%#v)", len(result.BindHosts), result.BindHosts)
	}
}

func startTestHTTPServer(t *testing.T, listeners []net.Listener) {
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

func requireHTTPReachable(t *testing.T, host string, port int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		err := httpGET(host, port)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %s:%d to be reachable: %v", host, port, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func requireHTTPUnreachable(t *testing.T, host string, port int) {
	t.Helper()

	if err := httpGET(host, port); err == nil {
		t.Fatalf("expected %s:%d to be unreachable", host, port)
	}
}

func httpGET(host string, port int) error {
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
