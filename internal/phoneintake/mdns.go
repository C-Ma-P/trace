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
			msg := fmt.Sprintf("mDNS: SetMulticastInterface on %s failed (non-fatal): %v", iface.Name, err)
			log.Printf("[phone-intake] %s", msg)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-socket-setup-failed", msg,
				map[string]any{"iface": iface.Name, "error": err.Error()}))
		}
		if err := pc.SetMulticastTTL(255); err != nil {
			msg := fmt.Sprintf("mDNS: SetMulticastTTL on %s failed (non-fatal): %v", iface.Name, err)
			log.Printf("[phone-intake] %s", msg)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-socket-setup-failed", msg,
				map[string]any{"iface": iface.Name, "error": err.Error()}))
		}
		if err := pc.SetMulticastLoopback(true); err != nil {
			msg := fmt.Sprintf("mDNS: SetMulticastLoopback on %s failed (non-fatal): %v", iface.Name, err)
			log.Printf("[phone-intake] %s", msg)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-socket-setup-failed", msg,
				map[string]any{"iface": iface.Name, "error": err.Error()}))
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
			msg := fmt.Sprintf("mDNS: initial announce on %s failed: %v", b.iface.Name, err)
			log.Printf("[phone-intake] %s", msg)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-announce-failed", msg,
				map[string]any{"iface": b.iface.Name, "error": err.Error()}))
		}
	}

	// Responder goroutine per interface.
	for _, b := range active {
		b := b
		ip6 := interfaceRoutableIPv6Addrs(b.iface)
		go mdnsRespond(ctx, b.conn, ip4, ip6, emit)
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
	// the mDNS stack as healthy.  We run three checks in sequence:
	//   1. multicast path    — standard RFC 6762 multicast loopback (A)
	//   2. one-shot A path   — Android-style legacy query from an ephemeral port
	//   3. one-shot AAAA path — AAAA query; expects positive AAAA or NSEC negative
	go func() {
		time.Sleep(mdnsCheckDelay)
		verifyMDNSResolution(lanIP, active[0].iface, emit)
		verifyMDNSOneShotResolution(lanIP, emit)
		verifyMDNSOneShotAAAAResolution(lanIP, emit)
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

// mdnsRespond reads mDNS queries from conn and replies for stableHostname.
//
// Per RFC 6762 §6.7 it distinguishes three query classes:
//   - QU (Query Unicast): QU bit set in Qclass → unicast reply, mDNS format
//   - legacy/one-shot:    source port ≠ 5353 → unicast reply, DNS format (question echoed)
//   - normal mDNS:        source port = 5353, no QU bit → multicast reply, mDNS format
//
// It handles A, ANY, and AAAA queries for stableHostname.  For AAAA queries on
// a host with no routable IPv6 address, it returns an NSEC negative response
// per RFC 6762 §6.1 so Android resolvers can promptly fall back to A.
func mdnsRespond(ctx context.Context, conn *net.UDPConn, ip4 net.IP, ip6Addrs []net.IP, emit activity.Emitter) {
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
		quBitSet := false
		for _, q := range query.Question {
			// The high bit of Qclass is the QU (Query Unicast) flag in mDNS.
			const quBit = 0x8000
			if q.Qclass&quBit != 0 {
				quBitSet = true
			}
			// Compare without trailing dot.
			name := q.Name
			if len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			if name != stableHostname {
				continue
			}
			switch q.Qtype {
			case dns.TypeA:
				msg := fmt.Sprintf("mDNS: A query from %s:%d", src.IP, src.Port)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-query", msg,
					map[string]any{"qtype": "A", "from": fmt.Sprintf("%s:%d", src.IP, src.Port)}))
				answers = append(answers, aRec)
			case dns.TypeANY:
				msg := fmt.Sprintf("mDNS: ANY query from %s:%d", src.IP, src.Port)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-query", msg,
					map[string]any{"qtype": "ANY", "from": fmt.Sprintf("%s:%d", src.IP, src.Port)}))
				answers = append(answers, aRec)
				for _, ip6 := range ip6Addrs {
					answers = append(answers, buildAAAARecord(ip6))
				}
			case dns.TypeAAAA:
				if len(ip6Addrs) > 0 {
					msg := fmt.Sprintf("mDNS: AAAA query from %s:%d → positive (%d addr)", src.IP, src.Port, len(ip6Addrs))
					log.Printf("[phone-intake] %s", msg)
					emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-query", msg,
						map[string]any{"qtype": "AAAA", "from": fmt.Sprintf("%s:%d", src.IP, src.Port)}))
					for _, ip6 := range ip6Addrs {
						answers = append(answers, buildAAAARecord(ip6))
					}
				} else {
					// RFC 6762 §6.1: send NSEC to signal authoritative "no AAAA" so
					// the resolver does not have to wait for the full negative TTL.
					msg := fmt.Sprintf("mDNS: AAAA query from %s:%d → NSEC negative (no IPv6)", src.IP, src.Port)
					log.Printf("[phone-intake] %s", msg)
					emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-query", msg,
						map[string]any{"qtype": "AAAA", "from": fmt.Sprintf("%s:%d", src.IP, src.Port), "nsec": true}))
					answers = append(answers, buildNSECRecord())
				}
			default:
				msg := fmt.Sprintf("mDNS: qtype %s from %s:%d — ignored", dns.TypeToString[q.Qtype], src.IP, src.Port)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-query", msg,
					map[string]any{"qtype": dns.TypeToString[q.Qtype], "from": fmt.Sprintf("%s:%d", src.IP, src.Port)}))
			}
		}
		if len(answers) == 0 {
			continue
		}

		// Legacy/one-shot: any query arriving from a source port other than 5353.
		// RFC 6762 §6.7: "If the source UDP port in a received Multicast DNS query
		// is not port 5353, this indicates that the querier will not be listening
		// for Multicast DNS responses, and that therefore all responses MUST be
		// sent via normal (unicast) DNS to the querier's port."
		isLegacy := src.Port != mdnsMcastPort

		resp := new(dns.Msg)
		resp.SetReply(&query)
		resp.Response = true
		resp.Authoritative = true
		resp.RecursionDesired = false
		resp.RecursionAvailable = false
		resp.Answer = answers
		if !isLegacy {
			// RFC 6762 §18.14: mDNS responses MUST NOT echo the question section
			// except in legacy unicast responses (where it MUST be echoed).
			resp.Question = nil
		}

		packed, err := resp.Pack()
		if err != nil {
			continue
		}

		switch {
		case quBitSet:
			// RFC 6762 §6.7: unicast reply for QU queries; mDNS wire format.
			if _, err := conn.WriteToUDP(packed, src); err != nil {
				msg := fmt.Sprintf("mDNS: unicast write error to %s: %v", src, err)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-write-error", msg,
					map[string]any{"dst": src.String(), "error": err.Error()}))
			}
		case isLegacy:
			// RFC 6762 §6.7: legacy/one-shot query → unicast reply with question echoed.
			if _, err := conn.WriteToUDP(packed, src); err != nil {
				msg := fmt.Sprintf("mDNS: unicast write error to %s: %v", src, err)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-write-error", msg,
					map[string]any{"dst": src.String(), "error": err.Error()}))
			}
		default:
			// Normal mDNS query from port 5353 → multicast reply.
			if _, err := conn.WriteToUDP(packed, mdnsGroupUDPAddr); err != nil {
				msg := fmt.Sprintf("mDNS: multicast write error: %v", err)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-write-error", msg,
					map[string]any{"error": err.Error()}))
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

// verifyMDNSOneShotResolution performs a self-check that simulates Android's
// "one-shot" / legacy mDNS behavior (RFC 6762 §5.1): it sends an A-query from
// an ephemeral (non-5353) source port to the mDNS multicast address and waits
// for a unicast reply back to that ephemeral port.  This path exercises the
// legacy/one-shot branch of mdnsRespond and is the path most Android devices
// take via getaddrinfo().
func verifyMDNSOneShotResolution(lanIP string, emit activity.Emitter) {
	// Bind to an ephemeral port on the LAN IP — not port 5353.  This is what
	// Android's getaddrinfo() does when performing a "one-shot" mDNS query.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(lanIP), Port: 0})
	if err != nil {
		msg := fmt.Sprintf("mDNS one-shot self-check: cannot open socket: %v", err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-oneshot-check-failed", msg,
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}
	defer conn.Close()

	// Build a standard mDNS query (no QU bit, ID = 0).
	query := new(dns.Msg)
	query.SetQuestion(stableHostname+".", dns.TypeA)
	query.Id = 0
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-oneshot-check-failed",
			"mDNS one-shot self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}

	// Send the query to the mDNS multicast address from our ephemeral port.
	// The responder will see source port ≠ 5353 and send a unicast reply back.
	if _, err := conn.WriteToUDP(packed, mdnsGroupUDPAddr); err != nil {
		msg := fmt.Sprintf("mDNS one-shot self-check: failed to send query: %v", err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-oneshot-check-failed", msg,
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}

	deadline := time.Now().Add(mdnsCheckTimeout)
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
			if name != stableHostname {
				continue
			}
			if a.A.Equal(net.ParseIP(lanIP)) {
				msg := fmt.Sprintf("mDNS one-shot self-check: %s → %s ✓ (unicast reply received)", stableHostname, a.A)
				log.Printf("[phone-intake] %s", msg)
				emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-oneshot-check-ok", msg,
					map[string]any{"ip": lanIP, "resolvedIP": a.A.String()}))
				return
			}
			badAnswerIP = a.A
		}
	}

	if badAnswerIP != nil {
		msg := fmt.Sprintf(
			"mDNS one-shot self-check: %s resolved to %s (want %s) — possible mDNS conflict",
			stableHostname, badAnswerIP, lanIP,
		)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-oneshot-check-bad-answer", msg,
			map[string]any{"ip": lanIP, "resolvedIP": badAnswerIP.String()}))
		return
	}

	msg := fmt.Sprintf(
		"mDNS one-shot self-check: no unicast reply for %s within %s — "+
			"Android devices may not resolve %s on this network",
		stableHostname, mdnsCheckTimeout, stableHostname,
	)
	log.Printf("[phone-intake] %s", msg)
	emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-oneshot-check-timeout", msg,
		map[string]any{"ip": lanIP}))
}

// verifyMDNSOneShotAAAAResolution performs a one-shot AAAA self-check.  It
// sends an AAAA query from an ephemeral port (Android-style) and considers the
// check successful if:
//   - a valid AAAA record is returned (host has routable IPv6), or
//   - a standards-compliant NSEC negative response is returned (no IPv6, which
//     is the expected outcome on an IPv4-only LAN).
func verifyMDNSOneShotAAAAResolution(lanIP string, emit activity.Emitter) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(lanIP), Port: 0})
	if err != nil {
		msg := fmt.Sprintf("mDNS AAAA self-check: cannot open socket: %v", err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-aaaa-check-failed", msg,
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}
	defer conn.Close()

	query := new(dns.Msg)
	query.SetQuestion(stableHostname+".", dns.TypeAAAA)
	query.Id = 0
	query.RecursionDesired = false

	packed, err := query.Pack()
	if err != nil {
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-aaaa-check-failed",
			"mDNS AAAA self-check: failed to build query",
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}

	if _, err := conn.WriteToUDP(packed, mdnsGroupUDPAddr); err != nil {
		msg := fmt.Sprintf("mDNS AAAA self-check: failed to send query: %v", err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-aaaa-check-failed", msg,
			map[string]any{"ip": lanIP, "error": err.Error()}))
		return
	}

	deadline := time.Now().Add(mdnsCheckTimeout)
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
				if name == stableHostname {
					msg := fmt.Sprintf("mDNS AAAA self-check: %s → %s ✓ (AAAA answer)", stableHostname, rec.AAAA)
					log.Printf("[phone-intake] %s", msg)
					emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-aaaa-check-ok", msg,
						map[string]any{"ip": lanIP, "ipv6": rec.AAAA.String()}))
					return
				}
			case *dns.NSEC:
				name := rec.Hdr.Name
				if len(name) > 0 && name[len(name)-1] == '.' {
					name = name[:len(name)-1]
				}
				if name != stableHostname {
					continue
				}
				// NSEC with A in bitmap but not AAAA → compliant negative response.
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
					msg := fmt.Sprintf("mDNS AAAA self-check: %s → NSEC negative (A-only host) ✓", stableHostname)
					log.Printf("[phone-intake] %s", msg)
					emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-aaaa-check-nsec", msg,
						map[string]any{"ip": lanIP}))
					return
				}
			}
		}
	}

	msg := fmt.Sprintf("mDNS AAAA self-check: no response for %s within %s", stableHostname, mdnsCheckTimeout)
	log.Printf("[phone-intake] %s", msg)
	emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-aaaa-check-timeout", msg,
		map[string]any{"ip": lanIP}))
}

// buildAAAARecord returns an mDNS AAAA record for stableHostname with the
// cache-flush bit set (RFC 6762 §11.3).
func buildAAAARecord(ip6 net.IP) *dns.AAAA {
	return &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   stableHostname + ".",
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET | 0x8000, // cache-flush bit
			Ttl:    mdnsHostTTL,
		},
		AAAA: ip6,
	}
}

// buildNSECRecord returns an mDNS NSEC record for stableHostname indicating
// that only the A record type exists (not AAAA).  This is the RFC 6762 §6.1
// negative response mechanism for unique-record owners.
func buildNSECRecord() *dns.NSEC {
	return &dns.NSEC{
		Hdr: dns.RR_Header{
			Name:   stableHostname + ".",
			Rrtype: dns.TypeNSEC,
			Class:  dns.ClassINET | 0x8000, // cache-flush bit
			Ttl:    mdnsHostTTL,
		},
		NextDomain: stableHostname + ".",
		TypeBitMap: []uint16{dns.TypeA}, // A exists; AAAA does not
	}
}

// interfaceRoutableIPv6Addrs returns the non-link-local, non-loopback IPv6
// addresses assigned to iface.  These are the addresses worth advertising in
// AAAA records; link-local addresses (fe80::/10) are excluded because they
// require a zone ID that browsers cannot embed in a URL.
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
				continue // skip IPv4
			}
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			result = append(result, ip)
		}
	}
	return result
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
