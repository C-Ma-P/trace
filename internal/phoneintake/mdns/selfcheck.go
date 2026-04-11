package mdns

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
)

func verifyMDNSResolution(hostname, lanIP string, checkIface net.Interface, hooks Hooks) {
	conn, err := net.ListenMulticastUDP("udp4", &checkIface, mcastGroupAddr)
	if err != nil {
		hooks.warn("mdns-check-failed",
			fmt.Sprintf("mDNS self-check: cannot open socket on %s: %v", checkIface.Name, err),
			map[string]any{"ip": lanIP, "iface": checkIface.Name, "error": err.Error()})
		return
	}
	defer conn.Close()

	checkPC := ipv4.NewPacketConn(conn)
	if err := checkPC.SetMulticastInterface(&checkIface); err != nil {
		log.Printf("[phone-intake] mDNS self-check: SetMulticastInterface failed (non-fatal): %v", err)
	}
	if err := checkPC.SetMulticastLoopback(true); err != nil {
		log.Printf("[phone-intake] mDNS self-check: SetMulticastLoopback failed (non-fatal): %v", err)
	}
	if err := checkPC.SetMulticastTTL(255); err != nil {
		log.Printf("[phone-intake] mDNS self-check: SetMulticastTTL failed (non-fatal): %v", err)
	}

	query := new(dns.Msg)
	query.SetQuestion(hostname+".", dns.TypeA)
	query.Id = 0
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		hooks.warn("mdns-check-send-failed",
			"mDNS self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	if _, err := conn.WriteToUDP(packed, mcastGroupAddr); err != nil {
		hooks.warn("mdns-check-send-failed",
			fmt.Sprintf("mDNS self-check: failed to send query on %s: %v", checkIface.Name, err),
			map[string]any{"ip": lanIP, "iface": checkIface.Name, "error": err.Error()})
		return
	}

	deadline := time.Now().Add(checkTimeout)
	buf := make([]byte, 65535)
	var badAnswerIP net.IP
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var resp dns.Msg
		if err := resp.Unpack(buf[:n]); err != nil {
			continue
		}
		if !resp.Response {
			continue
		}
		for _, rr := range resp.Answer {
			a, ok := rr.(*dns.A)
			if !ok {
				continue
			}
			name := a.Hdr.Name
			if len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			if name != hostname {
				continue
			}
			if a.A.Equal(net.ParseIP(lanIP)) {
				hooks.success("mdns-check-ok",
					fmt.Sprintf("mDNS: %s → %s ✓", hostname, a.A),
					map[string]any{"ip": lanIP, "iface": checkIface.Name, "resolvedIP": a.A.String()})
				return
			}
			badAnswerIP = a.A
		}
	}

	if badAnswerIP != nil {
		hooks.warn("mdns-check-bad-answer",
			fmt.Sprintf("mDNS self-check: %s resolved to %s (want %s) — possible mDNS conflict on the network", hostname, badAnswerIP, lanIP),
			map[string]any{"ip": lanIP, "iface": checkIface.Name, "resolvedIP": badAnswerIP.String()})
		return
	}

	hooks.warn("mdns-check-timeout",
		fmt.Sprintf("mDNS self-check: no response for %s within %s — trace.local may not be reachable from phones on this network", hostname, checkTimeout),
		map[string]any{"ip": lanIP, "iface": checkIface.Name})
}

func verifyOneShotResolution(hostname, lanIP string, hooks Hooks) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(lanIP), Port: 0})
	if err != nil {
		hooks.warn("mdns-oneshot-check-failed",
			fmt.Sprintf("mDNS one-shot self-check: cannot open socket: %v", err),
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}
	defer conn.Close()

	queryID := dns.Id()
	query := new(dns.Msg)
	query.SetQuestion(hostname+".", dns.TypeA)
	query.Id = queryID
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		hooks.warn("mdns-oneshot-check-failed",
			"mDNS one-shot self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	if _, err := conn.WriteToUDP(packed, mcastGroupAddr); err != nil {
		hooks.warn("mdns-oneshot-check-failed",
			fmt.Sprintf("mDNS one-shot self-check: failed to send query: %v", err),
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	deadline := time.Now().Add(checkTimeout)
	buf := make([]byte, 65535)
	var badAnswerIP net.IP
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var resp dns.Msg
		if err := resp.Unpack(buf[:n]); err != nil {
			continue
		}
		if !resp.Response {
			continue
		}
		for _, rr := range resp.Answer {
			a, ok := rr.(*dns.A)
			if !ok {
				continue
			}
			name := a.Hdr.Name
			if len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			if name != hostname {
				continue
			}
			if a.A.Equal(net.ParseIP(lanIP)) {
				var violations []string
				if resp.Id != queryID {
					violations = append(violations,
						fmt.Sprintf("ID mismatch: got %d want %d", resp.Id, queryID))
				}
				if len(resp.Question) == 0 {
					violations = append(violations, "question section absent")
				}
				if a.Hdr.Class&0x8000 != 0 {
					violations = append(violations, "cache-flush bit set (must be clear)")
				}
				if a.Hdr.Ttl > legacyMaxTTL {
					violations = append(violations,
						fmt.Sprintf("TTL %d > %d s limit", a.Hdr.Ttl, legacyMaxTTL))
				}
				if len(violations) > 0 {
					hooks.warn("mdns-oneshot-check-rfc-violation",
						fmt.Sprintf("mDNS one-shot self-check: %s → %s resolved but reply violates RFC6762§6.7: %v", hostname, a.A, violations),
						map[string]any{"ip": lanIP, "resolvedIP": a.A.String(), "violations": violations})
					return
				}
				hooks.success("mdns-oneshot-check-ok",
					fmt.Sprintf("mDNS: one-shot %s → %s ✓ (RFC6762§6.7, TTL=%ds)", hostname, a.A, a.Hdr.Ttl),
					map[string]any{"ip": lanIP, "resolvedIP": a.A.String(), "ttl": a.Hdr.Ttl})
				return
			}
			badAnswerIP = a.A
		}
	}

	if badAnswerIP != nil {
		hooks.warn("mdns-oneshot-check-bad-answer",
			fmt.Sprintf("mDNS one-shot self-check: %s resolved to %s (want %s) — possible mDNS conflict", hostname, badAnswerIP, lanIP),
			map[string]any{"ip": lanIP, "resolvedIP": badAnswerIP.String()})
		return
	}

	hooks.warn("mdns-oneshot-check-timeout",
		fmt.Sprintf("mDNS one-shot self-check: no unicast reply for %s within %s — Android devices may not resolve %s on this network", hostname, checkTimeout, hostname),
		map[string]any{"ip": lanIP})
}

func verifyOneShotAAAAResolution(hostname, lanIP string, hooks Hooks) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(lanIP), Port: 0})
	if err != nil {
		hooks.warn("mdns-aaaa-check-failed",
			fmt.Sprintf("mDNS AAAA self-check: cannot open socket: %v", err),
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}
	defer conn.Close()

	query := new(dns.Msg)
	query.SetQuestion(hostname+".", dns.TypeAAAA)
	query.Id = 0
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		hooks.warn("mdns-aaaa-check-failed",
			"mDNS AAAA self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	if _, err := conn.WriteToUDP(packed, mcastGroupAddr); err != nil {
		hooks.warn("mdns-aaaa-check-failed",
			fmt.Sprintf("mDNS AAAA self-check: failed to send query: %v", err),
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	deadline := time.Now().Add(checkTimeout)
	buf := make([]byte, 65535)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var resp dns.Msg
		if err := resp.Unpack(buf[:n]); err != nil {
			continue
		}
		if !resp.Response {
			continue
		}
		for _, rr := range resp.Answer {
			switch rec := rr.(type) {
			case *dns.AAAA:
				name := rec.Hdr.Name
				if len(name) > 0 && name[len(name)-1] == '.' {
					name = name[:len(name)-1]
				}
				if name == hostname {
					hooks.success("mdns-aaaa-check-ok",
						fmt.Sprintf("mDNS: AAAA self-check %s → %s ✓", hostname, rec.AAAA),
						map[string]any{"ip": lanIP, "ipv6": rec.AAAA.String()})
					return
				}
			case *dns.NSEC:
				name := rec.Hdr.Name
				if len(name) > 0 && name[len(name)-1] == '.' {
					name = name[:len(name)-1]
				}
				if name != hostname {
					continue
				}
				hasA, hasAAAA := false, false
				for _, t := range rec.TypeBitMap {
					if t == dns.TypeA {
						hasA = true
					}
					if t == dns.TypeAAAA {
						hasAAAA = true
					}
				}
				if hasA && !hasAAAA {
					hooks.success("mdns-aaaa-check-nsec",
						fmt.Sprintf("mDNS: AAAA self-check %s → NSEC (no IPv6) ✓", hostname),
						map[string]any{"ip": lanIP})
					return
				}
			}
		}
	}

	hooks.warn("mdns-aaaa-check-timeout",
		fmt.Sprintf("mDNS AAAA self-check: no response for %s within %s", hostname, checkTimeout),
		map[string]any{"ip": lanIP})
}

func verifyOneShotHTTPSResolution(hostname, lanIP string, hooks Hooks) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(lanIP), Port: 0})
	if err != nil {
		hooks.warn("mdns-https-check-failed",
			fmt.Sprintf("mDNS HTTPS self-check: cannot open socket: %v", err),
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}
	defer conn.Close()

	query := new(dns.Msg)
	query.SetQuestion(hostname+".", dns.TypeHTTPS)
	query.Id = 0
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		hooks.warn("mdns-https-check-failed",
			"mDNS HTTPS self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	if _, err := conn.WriteToUDP(packed, mcastGroupAddr); err != nil {
		hooks.warn("mdns-https-check-failed",
			fmt.Sprintf("mDNS HTTPS self-check: failed to send query: %v", err),
			map[string]any{"ip": lanIP, "error": err.Error()})
		return
	}

	deadline := time.Now().Add(checkTimeout)
	buf := make([]byte, 65535)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		var resp dns.Msg
		if err := resp.Unpack(buf[:n]); err != nil {
			continue
		}
		if !resp.Response {
			continue
		}
		for _, rr := range resp.Answer {
			nsec, ok := rr.(*dns.NSEC)
			if !ok {
				continue
			}
			name := nsec.Hdr.Name
			if len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			if name != hostname {
				continue
			}
			hasHTTPS := false
			for _, t := range nsec.TypeBitMap {
				if t == dns.TypeHTTPS {
					hasHTTPS = true
				}
			}
			if !hasHTTPS {
				hooks.success("mdns-https-check-nsec",
					fmt.Sprintf("mDNS: HTTPS self-check %s → NSEC (no HTTPS RR) ✓", hostname),
					map[string]any{"ip": lanIP})
				return
			}
		}
	}

	hooks.warn("mdns-https-check-timeout",
		fmt.Sprintf("mDNS HTTPS self-check: no NSEC response for %s within %s — Chrome may stall on TYPE_HTTPS lookup", hostname, checkTimeout),
		map[string]any{"ip": lanIP})
}
