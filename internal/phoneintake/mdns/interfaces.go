package mdns

import (
	"fmt"
	"net"
	"strings"
)

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

func isVirtualInterface(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func eligibleLANInterfaces() []net.Interface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var result []net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		hasIPv4 := false
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				hasIPv4 = true
				break
			}
		}
		if hasIPv4 {
			result = append(result, iface)
		}
	}
	return result
}

func lanInterfaceForIP(lanIP string) (*net.Interface, error) {
	ip := net.ParseIP(lanIP)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP %q", lanIP)
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("listing interfaces: %w", err)
	}
	for i := range ifaces {
		addrs, err := ifaces[i].Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var candidate net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				candidate = v.IP
			case *net.IPAddr:
				candidate = v.IP
			}
			if candidate != nil && candidate.Equal(ip) {
				return &ifaces[i], nil
			}
		}
	}
	return nil, fmt.Errorf("no interface found for IP %s", lanIP)
}

func interfaceRoutableIPv6Addrs(iface net.Interface) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	var result []net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			ip := ipnet.IP
			if ip.To4() != nil {
				continue
			}
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			result = append(result, ip)
		}
	}
	return result
}
