package config

import (
	"fmt"
	"net"
)

func (g GatewayConfig) ResolvedHost() (string, error) {
	switch g.Bind {
	case "", "all":
		return "0.0.0.0", nil
	case "local":
		return "127.0.0.1", nil
	case "tailnet":
		ip, err := findTailnetIPv4()
		if err != nil {
			return "", err
		}
		return ip, nil
	default:
		return "", fmt.Errorf("unknown gateway bind mode: %s", g.Bind)
	}
}

func (g GatewayConfig) ResolvedAddr() (string, error) {
	host, err := g.ResolvedHost()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, g.Port), nil
}

func findTailnetIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			if ip4.IsLoopback() {
				continue
			}
			// Support both 100.64.0.0/10 (Tailscale default) and 10.0.0.0/8 (user-requested range)
			if isCIDR(ip4, "100.64.0.0/10") || isCIDR(ip4, "10.0.0.0/8") {
				return ip4.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no tailnet IPv4 address found")
}

func isCIDR(ip net.IP, cidr string) bool {
	_, block, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return block.Contains(ip)
}
