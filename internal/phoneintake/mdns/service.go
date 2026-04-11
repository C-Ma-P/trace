package mdns

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

func Start(cfg Config) (func(), error) {
	ip4 := net.ParseIP(cfg.IPv4).To4()
	if ip4 == nil {
		msg := fmt.Sprintf("mDNS: %q is not a valid IPv4 address — %s will not resolve", cfg.IPv4, cfg.Hostname)
		cfg.Hooks.warn("mdns-no-interface", msg, map[string]any{"ip": cfg.IPv4})
		return func() {}, fmt.Errorf("mdns: %w", fmt.Errorf("invalid IPv4 address %q", cfg.IPv4))
	}

	ifaces := eligibleLANInterfaces()
	if len(ifaces) == 0 {
		iface, err := lanInterfaceForIP(cfg.IPv4)
		if err != nil {
			msg := fmt.Sprintf("mDNS: no eligible LAN interface found (%v) — %s will not resolve", err, cfg.Hostname)
			cfg.Hooks.warn("mdns-no-interface", msg, map[string]any{"ip": cfg.IPv4, "error": err.Error()})
			return func() {}, fmt.Errorf("mdns: no eligible interface: %w", err)
		}
		ifaces = []net.Interface{*iface}
	}

	ifaceNames := make([]string, len(ifaces))
	for i, iface := range ifaces {
		ifaceNames[i] = iface.Name
	}
	cfg.Hooks.info("mdns-starting",
		fmt.Sprintf("mDNS: starting on interfaces %v for %s → %s", ifaceNames, cfg.Hostname, cfg.IPv4),
		map[string]any{"ip": cfg.IPv4, "ifaces": ifaceNames})

	type bound struct {
		iface net.Interface
		conn  *net.UDPConn
	}
	var active []bound
	for _, iface := range ifaces {
		iface := iface
		conn, err := net.ListenMulticastUDP("udp4", &iface, mcastGroupAddr)
		if err != nil {
			log.Printf("[phone-intake] mDNS: multicast join failed on %s: %v", iface.Name, err)
			continue
		}
		pc := ipv4.NewPacketConn(conn)
		if err := pc.SetMulticastInterface(&iface); err != nil {
			cfg.Hooks.warn("mdns-socket-setup-failed",
				fmt.Sprintf("mDNS: SetMulticastInterface on %s failed (non-fatal): %v", iface.Name, err),
				map[string]any{"iface": iface.Name, "error": err.Error()})
		}
		if err := pc.SetMulticastTTL(255); err != nil {
			cfg.Hooks.warn("mdns-socket-setup-failed",
				fmt.Sprintf("mDNS: SetMulticastTTL on %s failed (non-fatal): %v", iface.Name, err),
				map[string]any{"iface": iface.Name, "error": err.Error()})
		}
		if err := pc.SetMulticastLoopback(true); err != nil {
			cfg.Hooks.warn("mdns-socket-setup-failed",
				fmt.Sprintf("mDNS: SetMulticastLoopback on %s failed (non-fatal): %v", iface.Name, err),
				map[string]any{"iface": iface.Name, "error": err.Error()})
		}
		active = append(active, bound{iface, conn})
	}

	if len(active) == 0 {
		cfg.Hooks.warn("mdns-bind-failed",
			fmt.Sprintf("mDNS: failed to join multicast on any interface — %s will not resolve", cfg.Hostname),
			map[string]any{"ip": cfg.IPv4, "ifaces": ifaceNames})
		return func() {}, fmt.Errorf("mdns: failed to join multicast on any interface")
	}

	activeNames := make([]string, len(active))
	for i, b := range active {
		activeNames[i] = b.iface.Name
	}
	cfg.Hooks.success("mdns-advertisement-active",
		fmt.Sprintf("mDNS: advertising %s → %s on interfaces %v", cfg.Hostname, cfg.IPv4, activeNames),
		map[string]any{"ip": cfg.IPv4, "ifaces": activeNames})

	ctx, cancel := context.WithCancel(context.Background())

	aRec := buildARecord(cfg.Hostname, ip4)

	for _, b := range active {
		if err := sendAnnounce(b.conn, aRec); err != nil {
			cfg.Hooks.warn("mdns-announce-failed",
				fmt.Sprintf("mDNS: initial announce on %s failed: %v", b.iface.Name, err),
				map[string]any{"iface": b.iface.Name, "error": err.Error()})
		}
	}

	for _, b := range active {
		b := b
		ip6 := interfaceRoutableIPv6Addrs(b.iface)
		go respond(ctx, cfg.Hostname, b.conn, ip4, ip6, cfg.Hooks)
	}

	go func() {
		t := time.NewTicker(announceInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, b := range active {
					_ = sendAnnounce(b.conn, aRec)
				}
			}
		}
	}()

	checkIface := active[0].iface
	hooks := cfg.Hooks
	hostname := cfg.Hostname
	lanIP := cfg.IPv4
	go func() {
		time.Sleep(checkDelay)
		verifyMDNSResolution(hostname, lanIP, checkIface, hooks)
		verifyOneShotResolution(hostname, lanIP, hooks)
		verifyOneShotAAAAResolution(hostname, lanIP, hooks)
		verifyOneShotHTTPSResolution(hostname, lanIP, hooks)
	}()

	return func() {
		cancel()
		for _, b := range active {
			_ = b.conn.Close()
		}
		cfg.Hooks.info("mdns-stopped", "mDNS: stopped", nil)
	}, nil
}
