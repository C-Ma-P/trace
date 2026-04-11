package phoneintake

import (
	"net"
	"os"
	"path/filepath"
	"strings"
)

// virtualPrefixes lists interface name prefixes that indicate virtual/non-LAN
// interfaces. Matching is a case-insensitive prefix match.
var virtualPrefixes = []string{
	"docker",
	"br-",
	"veth",
	"virbr",
	"vmnet",
	"vboxnet",
	"tailscale",
	"zt",
	"tun",
	"tap",
}

// HostSelection holds the result of LAN host detection for phone intake.
type HostSelection struct {
	Host   string `json:"host"`   // chosen IP or hostname
	Iface  string `json:"iface"`  // interface name (empty for override or fallback)
	Source string `json:"source"` // "auto", "override", or "fallback"
}

// selectLANHost returns the best host for phone intake display and mDNS.
//
// If override is non-empty it is returned as-is with source "override".
// Otherwise, network interfaces are enumerated; loopback and virtual interfaces
// are excluded, and RFC1918 private-range addresses are preferred over other
// addresses. If no suitable interface is found, "localhost" is returned with
// source "fallback".
func selectLANHost(override string) HostSelection {
	if override != "" {
		return HostSelection{Host: override, Source: "override"}
	}

	type candidate struct {
		ip    net.IP
		iface string
		score int // higher = more preferred
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return HostSelection{Host: "localhost", Source: "fallback"}
	}

	var candidates []candidate
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			candidates = append(candidates, candidate{
				ip:    ip4,
				iface: iface.Name,
				score: rfc1918Score(ip4),
			})
		}
	}

	if len(candidates) == 0 {
		return HostSelection{Host: "localhost", Source: "fallback"}
	}

	// Pick the highest-scored candidate. On a tie, first-found wins (enumeration
	// order from the OS tends to prefer the primary interface).
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return HostSelection{Host: best.ip.String(), Iface: best.iface, Source: "auto"}
}

// isVirtualInterface reports whether the interface name matches a known
// virtual/non-LAN prefix.
func isVirtualInterface(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// rfc1918Score returns a preference score for a private-range IPv4 address.
// Higher score = more preferred LAN candidate.
//
//	192.168.x.x → 3  (most common home/office LAN)
//	10.x.x.x    → 2
//	172.16-31.x → 1
//	other       → 0
func rfc1918Score(ip net.IP) int {
	switch {
	case ip[0] == 192 && ip[1] == 168:
		return 3
	case ip[0] == 10:
		return 2
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return 1
	default:
		return 0
	}
}

const hostOverrideFile = "host-override"

// loadHostOverride reads the persisted display-host override from pkiDir.
// Returns "" if pkiDir is empty, the file is absent, or the content is blank.
func loadHostOverride(pkiDir string) string {
	if pkiDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(pkiDir, hostOverrideFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveHostOverride writes host as the persisted display-host override.
func saveHostOverride(pkiDir, host string) error {
	if pkiDir == "" {
		return nil // no persistent dir; accept silently
	}
	return os.WriteFile(filepath.Join(pkiDir, hostOverrideFile), []byte(host), 0o600)
}

// clearHostOverrideFile removes the persisted display-host override file.
func clearHostOverrideFile(pkiDir string) error {
	if pkiDir == "" {
		return nil
	}
	err := os.Remove(filepath.Join(pkiDir, hostOverrideFile))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
