package netbind

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

type DefaultMode int

const (
	DefaultLoopback DefaultMode = iota
	DefaultAny
)

type groupKind int

const (
	groupAdaptiveLoopback groupKind = iota
	groupAdaptiveAny
	groupExact
)

type exactBinding struct {
	host    string
	network string
	v6Only  bool
}

type bindGroup struct {
	kind      groupKind
	allowIPv4 bool
	allowIPv6 bool
	exact     exactBinding
}

type Plan struct {
	groups    []bindGroup
	ProbeHost string
}

type OpenResult struct {
	Listeners []net.Listener
	BindHosts []string
	Port      string
	ProbeHost string
}

type tokenKind int

const (
	tokenName tokenKind = iota
	tokenLocalhost
	tokenStar
	tokenIPv4
	tokenIPv6
	tokenIPv4Any
	tokenIPv6Any
)

type hostToken struct {
	kind      tokenKind
	canonical string
	key       string
}

var (
	ipFamiliesOnce sync.Once
	hasIPv4        bool
	hasIPv6        bool
)

func DetectIPFamilies() (bool, bool) {
	ipFamiliesOnce.Do(func() {
		if ips, err := net.LookupIP("localhost"); err == nil {
			for _, ip := range ips {
				if ip == nil {
					continue
				}
				if ip.To4() != nil {
					hasIPv4 = true
					continue
				}
				hasIPv6 = true
			}
		}

		if hasIPv4 && hasIPv6 {
			return
		}

		if addrs, err := net.InterfaceAddrs(); err == nil {
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if !ok || ipnet.IP == nil {
					continue
				}
				if ipnet.IP.To4() != nil {
					hasIPv4 = true
					continue
				}
				hasIPv6 = true
			}
		}
	})

	return hasIPv4, hasIPv6
}

func SelectAdaptiveLoopbackHost(hasIPv4, hasIPv6 bool) string {
	switch {
	case hasIPv4 && hasIPv6:
		return "localhost"
	case hasIPv6:
		return "::1"
	case hasIPv4:
		return "127.0.0.1"
	default:
		return "localhost"
	}
}

func SelectAdaptiveAnyHost(hasIPv4, hasIPv6 bool) string {
	switch {
	case hasIPv4 && hasIPv6:
		return "::"
	case hasIPv6:
		return "::"
	case hasIPv4:
		return "0.0.0.0"
	default:
		return "::"
	}
}

func ResolveAdaptiveLoopbackHost() string {
	hasIPv4, hasIPv6 := DetectIPFamilies()
	return SelectAdaptiveLoopbackHost(hasIPv4, hasIPv6)
}

func ResolveAdaptiveAnyHost() string {
	hasIPv4, hasIPv6 := DetectIPFamilies()
	return SelectAdaptiveAnyHost(hasIPv4, hasIPv6)
}

func IsLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func IsUnspecifiedHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsUnspecified()
}

func NormalizeHostInput(raw string) (string, error) {
	tokens, err := parseHostTokens(raw)
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, token.canonical)
	}
	return strings.Join(parts, ","), nil
}

func BuildPlan(raw string, defaultMode DefaultMode) (Plan, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return buildDefaultPlan(defaultMode), nil
	}

	tokens, err := parseHostTokens(raw)
	if err != nil {
		return Plan{}, err
	}

	for _, token := range tokens {
		if token.kind == tokenStar {
			return Plan{
				groups:    []bindGroup{{kind: groupAdaptiveAny}},
				ProbeHost: ResolveAdaptiveLoopbackHost(),
			}, nil
		}
	}

	hasIPv4Any := false
	hasIPv6Any := false
	for _, token := range tokens {
		switch token.kind {
		case tokenIPv4Any:
			hasIPv4Any = true
		case tokenIPv6Any:
			hasIPv6Any = true
		}
	}

	allowLocalhostIPv4 := !hasIPv4Any
	allowLocalhostIPv6 := !hasIPv6Any

	groups := make([]bindGroup, 0, len(tokens))
	seenExact := make(map[string]struct{}, len(tokens))
	addedLocalhost := false

	for _, token := range tokens {
		switch token.kind {
		case tokenLocalhost:
			if addedLocalhost || (!allowLocalhostIPv4 && !allowLocalhostIPv6) {
				continue
			}
			groups = append(groups, bindGroup{
				kind:      groupAdaptiveLoopback,
				allowIPv4: allowLocalhostIPv4,
				allowIPv6: allowLocalhostIPv6,
			})
			addedLocalhost = true
		case tokenIPv4Any:
			key := "exact:tcp4:0.0.0.0"
			if _, ok := seenExact[key]; ok {
				continue
			}
			seenExact[key] = struct{}{}
			groups = append(groups, bindGroup{
				kind: groupExact,
				exact: exactBinding{
					host:    "0.0.0.0",
					network: "tcp4",
				},
			})
		case tokenIPv6Any:
			key := "exact:tcp6:::"
			if _, ok := seenExact[key]; ok {
				continue
			}
			seenExact[key] = struct{}{}
			groups = append(groups, bindGroup{
				kind: groupExact,
				exact: exactBinding{
					host:    "::",
					network: "tcp6",
					v6Only:  true,
				},
			})
		case tokenIPv4:
			if hasIPv4Any {
				continue
			}
			key := "exact:tcp4:" + strings.ToLower(token.canonical)
			if _, ok := seenExact[key]; ok {
				continue
			}
			seenExact[key] = struct{}{}
			groups = append(groups, bindGroup{
				kind: groupExact,
				exact: exactBinding{
					host:    token.canonical,
					network: "tcp4",
				},
			})
		case tokenIPv6:
			if hasIPv6Any {
				continue
			}
			key := "exact:tcp6:" + strings.ToLower(token.canonical)
			if _, ok := seenExact[key]; ok {
				continue
			}
			seenExact[key] = struct{}{}
			groups = append(groups, bindGroup{
				kind: groupExact,
				exact: exactBinding{
					host:    token.canonical,
					network: "tcp6",
					v6Only:  true,
				},
			})
		case tokenName:
			key := "exact:tcp:" + token.key
			if _, ok := seenExact[key]; ok {
				continue
			}
			seenExact[key] = struct{}{}
			groups = append(groups, bindGroup{
				kind: groupExact,
				exact: exactBinding{
					host:    token.canonical,
					network: "tcp",
				},
			})
		}
	}

	plan := Plan{groups: groups}
	plan.ProbeHost = probeHostForGroups(groups)
	return plan, nil
}

func OpenPlan(plan Plan, port string) (OpenResult, error) {
	if port == "" {
		return OpenResult{}, errors.New("port cannot be empty")
	}

	selectedPort := port
	listeners := make([]net.Listener, 0, len(plan.groups))
	bindHosts := make([]string, 0, len(plan.groups))
	bindSeen := make(map[string]struct{}, len(plan.groups))

	closeAll := func() {
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}

	for _, group := range plan.groups {
		groupListeners, groupHosts, actualPort, err := openGroup(group, selectedPort)
		if err != nil {
			closeAll()
			return OpenResult{}, err
		}
		if selectedPort == "0" && actualPort != "" {
			selectedPort = actualPort
		}
		listeners = append(listeners, groupListeners...)
		for _, host := range groupHosts {
			key := strings.ToLower(host)
			if _, ok := bindSeen[key]; ok {
				continue
			}
			bindSeen[key] = struct{}{}
			bindHosts = append(bindHosts, host)
		}
	}

	return OpenResult{
		Listeners: listeners,
		BindHosts: bindHosts,
		Port:      selectedPort,
		ProbeHost: plan.ProbeHost,
	}, nil
}

func buildDefaultPlan(defaultMode DefaultMode) Plan {
	switch defaultMode {
	case DefaultAny:
		return Plan{
			groups:    []bindGroup{{kind: groupAdaptiveAny}},
			ProbeHost: ResolveAdaptiveLoopbackHost(),
		}
	default:
		return Plan{
			groups: []bindGroup{{
				kind:      groupAdaptiveLoopback,
				allowIPv4: true,
				allowIPv6: true,
			}},
			ProbeHost: ResolveAdaptiveLoopbackHost(),
		}
	}
}

func probeHostForGroups(groups []bindGroup) string {
	hasIPv4Any := false
	hasIPv6Any := false
	for _, group := range groups {
		if group.kind == groupAdaptiveLoopback {
			switch {
			case group.allowIPv4 && group.allowIPv6:
				return ResolveAdaptiveLoopbackHost()
			case group.allowIPv6:
				return "::1"
			case group.allowIPv4:
				return "127.0.0.1"
			}
		}
		if group.kind == groupAdaptiveAny {
			return ResolveAdaptiveLoopbackHost()
		}
		if group.kind != groupExact {
			continue
		}
		switch group.exact.host {
		case "0.0.0.0":
			hasIPv4Any = true
		case "::":
			hasIPv6Any = true
		}
	}

	switch {
	case hasIPv4Any && hasIPv6Any:
		return ResolveAdaptiveLoopbackHost()
	case hasIPv6Any:
		return "::1"
	case hasIPv4Any:
		return "127.0.0.1"
	}

	for _, group := range groups {
		if group.kind == groupExact {
			return group.exact.host
		}
	}
	return ResolveAdaptiveLoopbackHost()
}

func parseHostTokens(raw string) ([]hostToken, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("host cannot be empty")
	}

	parts := strings.Split(raw, ",")
	tokens := make([]hostToken, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		token, err := parseHostToken(part)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[token.key]; ok {
			continue
		}
		seen[token.key] = struct{}{}
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		return nil, errors.New("host cannot be empty")
	}

	return tokens, nil
}

func parseHostToken(raw string) (hostToken, error) {
	host := strings.TrimSpace(raw)
	if host == "" {
		return hostToken{}, errors.New("host list contains an empty entry")
	}

	if host == "*" {
		return hostToken{kind: tokenStar, canonical: "*", key: "*"}, nil
	}
	if strings.EqualFold(host, "localhost") {
		return hostToken{kind: tokenLocalhost, canonical: "localhost", key: "localhost"}, nil
	}

	trimmed := strings.Trim(host, "[]")
	if ip := net.ParseIP(trimmed); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			canonical := ip4.String()
			kind := tokenIPv4
			if ip4.IsUnspecified() {
				kind = tokenIPv4Any
			}
			return hostToken{kind: kind, canonical: canonical, key: canonical}, nil
		}

		canonical := ip.String()
		kind := tokenIPv6
		if ip.IsUnspecified() {
			kind = tokenIPv6Any
		}
		return hostToken{kind: kind, canonical: canonical, key: strings.ToLower(canonical)}, nil
	}

	return hostToken{
		kind:      tokenName,
		canonical: host,
		key:       strings.ToLower(host),
	}, nil
}

func openGroup(group bindGroup, port string) ([]net.Listener, []string, string, error) {
	switch group.kind {
	case groupAdaptiveLoopback:
		return openAdaptiveLoopbackGroup(group.allowIPv6, group.allowIPv4, port)
	case groupAdaptiveAny:
		return openAdaptiveAnyGroup(port)
	case groupExact:
		ln, actualPort, err := openExactListener(group.exact, port)
		if err != nil {
			return nil, nil, "", err
		}
		return []net.Listener{ln}, []string{group.exact.host}, actualPort, nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported bind group kind: %d", group.kind)
	}
}

func openAdaptiveLoopbackGroup(allowIPv6, allowIPv4 bool, port string) ([]net.Listener, []string, string, error) {
	if allowIPv6 && allowIPv4 {
		if ln6, actualPort, err6 := openExactListener(
			exactBinding{host: "::1", network: "tcp6", v6Only: true},
			port,
		); err6 == nil {
			if ln4, _, err4 := openExactListener(
				exactBinding{host: "127.0.0.1", network: "tcp4"},
				actualPort,
			); err4 == nil {
				return []net.Listener{ln6, ln4}, []string{"::1", "127.0.0.1"}, actualPort, nil
			}
			_ = ln6.Close()
		}
	}

	if allowIPv6 {
		ln6, actualPort, err := openExactListener(exactBinding{host: "::1", network: "tcp6", v6Only: true}, port)
		if err == nil {
			return []net.Listener{ln6}, []string{"::1"}, actualPort, nil
		}
	}

	if allowIPv4 {
		ln4, actualPort, err := openExactListener(exactBinding{host: "127.0.0.1", network: "tcp4"}, port)
		if err == nil {
			return []net.Listener{ln4}, []string{"127.0.0.1"}, actualPort, nil
		}
	}

	return nil, nil, "", fmt.Errorf("failed to open adaptive localhost listener on port %s", port)
}

func openAdaptiveAnyGroup(port string) ([]net.Listener, []string, string, error) {
	hasIPv4, hasIPv6 := DetectIPFamilies()

	if hasIPv4 && hasIPv6 {
		if ln6, actualPort, err6 := openExactListener(
			exactBinding{host: "::", network: "tcp6", v6Only: true},
			port,
		); err6 == nil {
			if ln4, _, err4 := openExactListener(
				exactBinding{host: "0.0.0.0", network: "tcp4"},
				actualPort,
			); err4 == nil {
				return []net.Listener{ln6, ln4}, []string{"::", "0.0.0.0"}, actualPort, nil
			}
			_ = ln6.Close()
		}
	}

	if hasIPv6 {
		ln6, actualPort, err := openExactListener(exactBinding{host: "::", network: "tcp6", v6Only: true}, port)
		if err == nil {
			return []net.Listener{ln6}, []string{"::"}, actualPort, nil
		}
	}

	if hasIPv4 {
		ln4, actualPort, err := openExactListener(exactBinding{host: "0.0.0.0", network: "tcp4"}, port)
		if err == nil {
			return []net.Listener{ln4}, []string{"0.0.0.0"}, actualPort, nil
		}
	}

	return nil, nil, "", fmt.Errorf("failed to open adaptive any-host listener on port %s", port)
}

func openExactListener(binding exactBinding, port string) (net.Listener, string, error) {
	listenConfig := net.ListenConfig{}
	if binding.network == "tcp6" && binding.v6Only {
		listenConfig.Control = applyIPv6OnlyControl(true)
	}

	ln, err := listenConfig.Listen(context.Background(), binding.network, net.JoinHostPort(binding.host, port))
	if err != nil {
		return nil, "", err
	}

	actualPort, err := listenerPort(ln)
	if err != nil {
		_ = ln.Close()
		return nil, "", err
	}

	return ln, actualPort, nil
}

func listenerPort(ln net.Listener) (string, error) {
	addr, ok := ln.Addr().(*net.TCPAddr)
	if ok {
		return strconv.Itoa(addr.Port), nil
	}

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return "", err
	}
	return port, nil
}
