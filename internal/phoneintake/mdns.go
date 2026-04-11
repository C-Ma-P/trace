package phoneintake

// mdns.go — RFC 6762 mDNS A-record responder for trace.local
//
// Design rationale
// ----------------
// Android and iOS resolve .local hostnames via mDNS (RFC 6762): the device
// sends a multicast A-query to 224.0.0.251:5353 and expects a direct A-record
// answer, not a DNS-SD service record as a side effect.
//
// The previous github.com/hashicorp/mdns implementation had two problems:
//   1. It bound to a single, pre-selected network interface; if the multicast
//      join failed silently on that interface (common on multi-homed hosts) the
//      responder was deaf.
//   2. It reported success as soon as the server *object* was created, with no
//      verification that trace.local was actually resolvable.
//
// This implementation fixes both issues:
//   • It opens a multicast socket on every eligible LAN interface (multicast-
//     capable, non-loopback, non-virtual, has an IPv4 address).  Failures on
//     individual interfaces are logged but do not block the others.
//   • After startup it performs a bounded self-check: it sends a real mDNS
//     A-query for trace.local on the LAN and waits for our own answer to come
//     back via multicast loopback.  The result is reported as a distinct
//     activity event so the UI can distinguish "server object created" from
//     "trace.local is actually resolvable".

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"

	"trace/internal/activity"
)

const (
	mdnsIPv4Addr         = "224.0.0.251"
	mdnsMcastPort        = 5353
	mdnsHostTTL          = 120 // seconds; per RFC 6762 §11.3
	mdnsAnnounceInterval = 25 * time.Second
	mdnsCheckDelay       = 750 * time.Millisecond
	mdnsCheckTimeout     = 3 * time.Second
)

var mdnsGroupUDPAddr = &net.UDPAddr{IP: net.ParseIP(mdnsIPv4Addr), Port: mdnsMcastPort}

// startMDNS starts a purpose-built mDNS A-record responder for stableHostname
// on all eligible LAN interfaces.  It returns a stop function.
//
// Successful startup means a multicast socket was opened on at least one
// interface AND (after a brief delay) a self-check query confirmed that
// trace.local resolves to lanIP on the LAN path.
func startMDNS(lanIP string, emit activity.Emitter) func() {
	ip4 := net.ParseIP(lanIP).To4()
	if ip4 == nil {
		msg := fmt.Sprintf("mDNS: %q is not a valid IPv4 address — %s will not resolve", lanIP, stableHostname)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-no-interface", msg,
			map[string]any{"ip": lanIP}))
		return func() {}
	}

	// Collect all eligible LAN interfaces; fall back to the one owning lanIP.
	ifaces := eligibleLANInterfaces()
	if len(ifaces) == 0 {
		iface, err := lanInterfaceForIP(lanIP)
		if err != nil {
			msg := fmt.Sprintf("mDNS: no eligible LAN interface found (%v) — %s will not resolve", err, stableHostname)
			log.Printf("[phone-intake] %s", msg)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-no-interface", msg,
				map[string]any{"ip": lanIP, "error": err.Error()}))
			return func() {}
		}
		ifaces = []net.Interface{*iface}
	}

	ifaceNames := make([]string, len(ifaces))
	for i, iface := range ifaces {
		ifaceNames[i] = iface.Name
	}
	log.Printf("[phone-intake] mDNS: starting on interfaces %v — advertising %s → %s", ifaceNames, stableHostname, lanIP)
	emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-starting",
		fmt.Sprintf("mDNS: starting on interfaces %v for %s → %s", ifaceNames, stableHostname, lanIP),
		map[string]any{"ip": lanIP, "ifaces": ifaceNames}))

	// Open a multicast receiver/sender per interface.
	type bound struct {
		iface net.Interface
		conn  *net.UDPConn
	}
	var active []bound
	for _, iface := range ifaces {
		iface := iface
		conn, err := net.ListenMulticastUDP("udp4", &iface, mdnsGroupUDPAddr)
		if err != nil {
			log.Printf("[phone-intake] mDNS: multicast join failed on %s: %v", iface.Name, err)
			continue
		}
		// Pin the outgoing multicast interface so responses leave from the right
		// source IP, set TTL=255 (required by RFC 6762 §11.4 for proper
		// link-local mDNS semantics), and enable loopback so the self-check
		// receives our own announcements on the same host.
		pc := ipv4.NewPacketConn(conn)
		if err := pc.SetMulticastInterface(&iface); err != nil {
			log.Printf("[phone-intake] mDNS: SetMulticastInterface on %s failed (non-fatal): %v", iface.Name, err)
		}
		if err := pc.SetMulticastTTL(255); err != nil {
			log.Printf("[phone-intake] mDNS: SetMulticastTTL on %s failed (non-fatal): %v", iface.Name, err)
		}
		if err := pc.SetMulticastLoopback(true); err != nil {
			log.Printf("[phone-intake] mDNS: SetMulticastLoopback on %s failed (non-fatal): %v", iface.Name, err)
		}
		active = append(active, bound{iface, conn})
	}

	if len(active) == 0 {
		msg := fmt.Sprintf("mDNS: failed to join multicast on any interface — %s will not resolve", stableHostname)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-bind-failed", msg,
			map[string]any{"ip": lanIP, "ifaces": ifaceNames}))
		return func() {}
	}

	activeNames := make([]string, len(active))
	for i, b := range active {
		activeNames[i] = b.iface.Name
	}
	log.Printf("[phone-intake] mDNS: advertising %s → %s on interfaces %v", stableHostname, lanIP, activeNames)
	emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-advertisement-active",
		fmt.Sprintf("mDNS: advertising %s → %s on interfaces %v", stableHostname, lanIP, activeNames),
		map[string]any{"ip": lanIP, "ifaces": activeNames}))

	ctx, cancel := context.WithCancel(context.Background())

	// Pre-build the A record used in all responses and announcements.
	aRec := buildARecord(ip4)

	// Gratuitous announcement on startup so listeners see us immediately.
	for _, b := range active {
		if err := sendMDNSAnnounce(b.conn, aRec); err != nil {
			log.Printf("[phone-intake] mDNS: initial announce on %s failed: %v", b.iface.Name, err)
		}
	}

	// Responder goroutine per interface.
	for _, b := range active {
		b := b
		go mdnsRespond(ctx, b.conn, ip4)
	}

	// Periodic re-announcement (keeps caches warm; RFC 6762 recommends ≤ TTL/2).
	go func() {
		t := time.NewTicker(mdnsAnnounceInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, b := range active {
					_ = sendMDNSAnnounce(b.conn, aRec)
				}
			}
		}
	}()

	// Self-check: verify that trace.local actually resolves before reporting
	// the mDNS stack as healthy.  We use the first active interface because
	// multicast loopback will echo the response back regardless.
	go func() {
		time.Sleep(mdnsCheckDelay)
		verifyMDNSResolution(lanIP, active[0].iface, emit)
	}()

	return func() {
		cancel()
		for _, b := range active {
			_ = b.conn.Close()
		}
		log.Printf("[phone-intake] mDNS: stopped")
		emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-stopped", "mDNS: stopped", nil))
	}
}

// mdnsRespond reads mDNS A/ANY queries from conn and replies for stableHostname.
func mdnsRespond(ctx context.Context, conn *net.UDPConn, ip4 net.IP) {
	buf := make([]byte, 65535)
	aRec := buildARecord(ip4)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// conn was closed
			return
		}

		var query dns.Msg
		if err := query.Unpack(buf[:n]); err != nil {
			continue
		}
		if query.Response {
			continue // ignore responses from other devices
		}

		var answers []dns.RR
		wantUnicast := false
		for _, q := range query.Question {
			if q.Qtype != dns.TypeA && q.Qtype != dns.TypeANY {
				continue
			}
			// The high bit of Qclass is the QU (Query Unicast) flag in mDNS.
			const quBit = 0x8000
			if q.Qclass&quBit != 0 {
				wantUnicast = true
			}
			// Compare without trailing dot.
			name := q.Name
			if len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			if name == stableHostname {
				answers = append(answers, aRec)
			}
		}
		if len(answers) == 0 {
			continue
		}

		resp := new(dns.Msg)
		resp.SetReply(&query)
		resp.Response = true
		resp.Authoritative = true
		resp.RecursionDesired = false
		resp.RecursionAvailable = false
		resp.Question = nil // mDNS responses MUST NOT echo the question section
		resp.Answer = answers

		packed, err := resp.Pack()
		if err != nil {
			continue
		}

		if wantUnicast {
			// RFC 6762 §6.7: respond unicast when the QU bit is set.
			if _, err := conn.WriteToUDP(packed, src); err != nil {
				log.Printf("[phone-intake] mDNS: unicast write error: %v", err)
			}
		} else {
			// Default: multicast so all listeners (including the querier) receive it.
			if _, err := conn.WriteToUDP(packed, mdnsGroupUDPAddr); err != nil {
				log.Printf("[phone-intake] mDNS: multicast write error: %v", err)
			}
		}
	}
}

// sendMDNSAnnounce sends an unsolicited (gratuitous) mDNS A-record announcement.
func sendMDNSAnnounce(conn *net.UDPConn, aRec dns.RR) error {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Answer = []dns.RR{aRec}
	packed, err := msg.Pack()
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(packed, mdnsGroupUDPAddr)
	return err
}

// buildARecord returns an mDNS A record for stableHostname with the cache-flush
// bit set (RFC 6762 §11.3 — required for unique records).
func buildARecord(ip4 net.IP) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   stableHostname + ".",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET | 0x8000, // 0x8000 = cache-flush bit
			Ttl:    mdnsHostTTL,
		},
		A: ip4,
	}
}

// verifyMDNSResolution performs a bounded self-check to confirm that
// stableHostname resolves via mDNS.  It sends a real mDNS A-query to the
// multicast group and waits for our own responder to echo back the answer
// (multicast loopback delivers it on the same interface).  The result is
// emitted as a distinct activity event.
func verifyMDNSResolution(lanIP string, checkIface net.Interface, emit activity.Emitter) {
	conn, err := net.ListenMulticastUDP("udp4", &checkIface, mdnsGroupUDPAddr)
	if err != nil {
		msg := fmt.Sprintf("mDNS self-check: cannot open socket on %s: %v", checkIface.Name, err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-check-failed", msg,
			map[string]any{"ip": lanIP, "iface": checkIface.Name, "error": err.Error()}))
		return
	}
	defer conn.Close()

	// Explicitly enable multicast loopback so we receive our own response on
	// the same host, and set TTL=255 consistent with the responder sockets.
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

	// Build a standard mDNS query (ID = 0, QU bit not set → expect multicast response).
	query := new(dns.Msg)
	query.SetQuestion(stableHostname+".", dns.TypeA)
	query.Id = 0 // RFC 6762 §18.1
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-check-send-failed",
			"mDNS self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}

	if _, err := conn.WriteToUDP(packed, mdnsGroupUDPAddr); err != nil {
		msg := fmt.Sprintf("mDNS self-check: failed to send query on %s: %v", checkIface.Name, err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-check-send-failed", msg,
			map[string]any{"ip": lanIP, "iface": checkIface.Name, "error": err.Error()}))
		return
	}

	deadline := time.Now().Add(mdnsCheckTimeout)
	buf := make([]byte, 65535)
	var badAnswerIP net.IP // non-nil when trace.local resolved but to the wrong address
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
			continue // skip queries we might hear
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
			if name != stableHostname {
				continue
			}
			if a.A.Equal(net.ParseIP(lanIP)) {
				msg := fmt.Sprintf("mDNS self-check: %s → %s ✓", stableHostname, a.A)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-check-ok", msg,
					map[string]any{"ip": lanIP, "iface": checkIface.Name, "resolvedIP": a.A.String()}))
				return
			}
			// Correct name but unexpected IP — remember and keep waiting in
			// case our own correct answer still arrives.
			badAnswerIP = a.A
		}
	}

	if badAnswerIP != nil {
		msg := fmt.Sprintf(
			"mDNS self-check: %s resolved to %s (want %s) — possible mDNS conflict on the network",
			stableHostname, badAnswerIP, lanIP,
		)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-check-bad-answer", msg,
			map[string]any{"ip": lanIP, "iface": checkIface.Name, "resolvedIP": badAnswerIP.String()}))
		return
	}

	msg := fmt.Sprintf(
		"mDNS self-check: no response for %s within %s — "+
			"trace.local may not be reachable from phones on this network",
		stableHostname, mdnsCheckTimeout,
	)
	log.Printf("[phone-intake] %s", msg)
	emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-check-timeout", msg,
		map[string]any{"ip": lanIP, "iface": checkIface.Name}))
}

// eligibleLANInterfaces returns all up, multicast-capable, non-loopback,
// non-virtual interfaces that have at least one IPv4 address.
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
			continue // mDNS requires multicast capability
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

// lanInterfaceForIP returns the network interface that owns the given IP.
// Used as a fallback when no eligible LAN interfaces are found by the broader
// enumeration.
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
