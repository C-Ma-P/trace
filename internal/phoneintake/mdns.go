package phoneintake

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/hashicorp/mdns"

	"trace/internal/activity"
)

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

func startMDNS(lanIP string, emit activity.Emitter) func() {
	iface, err := lanInterfaceForIP(lanIP)
	if err != nil {
		msg := fmt.Sprintf("mDNS: cannot resolve LAN interface (%v) — trace.local will not resolve", err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-no-interface", msg,
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return func() {}
	}

	log.Printf("[phone-intake] mDNS: selected interface %s (index %d) for LAN IP %s", iface.Name, iface.Index, lanIP)
	emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-starting",
		fmt.Sprintf("mDNS: starting on interface %s (index %d) for %s", iface.Name, iface.Index, lanIP),
		map[string]any{"ip": lanIP, "iface": iface.Name, "ifaceIndex": iface.Index}))

	svc, err := mdns.NewMDNSService(
		"trace",            // instance name
		"_trace._tcp",      // service type (carries A record for the hostname)
		"local.",           // domain
		stableHostname+".", // hostname → A record target preserving trace.local contract
		defaultPort,
		[]net.IP{net.ParseIP(lanIP)},
		nil, // no TXT records needed
	)
	if err != nil {
		msg := fmt.Sprintf("mDNS: service init failed (%v) — trace.local will not resolve", err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-start-failed", msg,
			map[string]any{"ip": lanIP, "iface": iface.Name, "ifaceIndex": iface.Index, "error": err.Error()}))
		return func() {}
	}

	srv, err := mdns.NewServer(&mdns.Config{
		Zone:   svc,
		Iface:  iface,
		Logger: log.New(io.Discard, "", 0), // structured events are emitted below; suppress library noise
	})
	if err != nil {
		msg := fmt.Sprintf("mDNS: bind/join failed on %s (%v) — trace.local will not resolve", iface.Name, err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-bind-failed", msg,
			map[string]any{"ip": lanIP, "iface": iface.Name, "ifaceIndex": iface.Index, "error": err.Error()}))
		return func() {}
	}

	log.Printf("[phone-intake] mDNS: advertising %s → %s on %s", stableHostname, lanIP, iface.Name)
	emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-started",
		fmt.Sprintf("mDNS: %s → %s (interface %s, index %d)", stableHostname, lanIP, iface.Name, iface.Index),
		map[string]any{"ip": lanIP, "iface": iface.Name, "ifaceIndex": iface.Index}))

	return func() {
		if shutdownErr := srv.Shutdown(); shutdownErr != nil {
			log.Printf("[phone-intake] mDNS: shutdown error: %v", shutdownErr)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-shutdown-error",
				fmt.Sprintf("mDNS: shutdown error: %v", shutdownErr),
				map[string]any{"error": shutdownErr.Error()}))
			return
		}
		log.Printf("[phone-intake] mDNS: stopped")
		emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-stopped", "mDNS: stopped", nil))
	}
}
