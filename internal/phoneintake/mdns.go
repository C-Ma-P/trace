package phoneintake

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"trace/internal/activity"
)

const (
	mdnsMulticastAddr = "224.0.0.251"
	mdnsPort          = 5353
	mdnsAnswerTTL     = 120
)

func lanInterfaceForIP(lanIP string) (*net.Interface, [4]byte, error) {
	var ipArr [4]byte
	ip := net.ParseIP(lanIP)
	if ip == nil {
		return nil, ipArr, fmt.Errorf("invalid IP %q", lanIP)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, ipArr, fmt.Errorf("IP %s is not IPv4", lanIP)
	}
	copy(ipArr[:], ip4)

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, ipArr, fmt.Errorf("listing interfaces: %w", err)
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
				return &ifaces[i], ipArr, nil
			}
		}
	}
	return nil, ipArr, fmt.Errorf("no interface found for IP %s", lanIP)
}

func startMDNS(lanIP string, emit activity.Emitter) func() {
	iface, ipArr, err := lanInterfaceForIP(lanIP)
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

	mcastAddr := &net.UDPAddr{IP: net.ParseIP(mdnsMulticastAddr), Port: mdnsPort}

	conn, err := net.ListenMulticastUDP("udp4", iface, mcastAddr)
	if err != nil {
		msg := fmt.Sprintf("mDNS: multicast bind/join failed on %s (%v) — trace.local will not resolve", iface.Name, err)
		log.Printf("[phone-intake] %s", msg)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-bind-failed", msg,
			map[string]any{"ip": lanIP, "iface": iface.Name, "ifaceIndex": iface.Index, "error": err.Error()}))
		return func() {}
	}

	log.Printf("[phone-intake] mDNS: listening — %s → %s on %s", stableHostname, lanIP, iface.Name)
	emit.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "mdns-started",
		fmt.Sprintf("mDNS: %s → %s (interface %s, index %d)", stableHostname, lanIP, iface.Name, iface.Index),
		map[string]any{"ip": lanIP, "iface": iface.Name, "ifaceIndex": iface.Index}))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		sendMDNSAnnouncement(conn, mcastAddr, ipArr)
	}()

	go runMDNSResponder(ctx, conn, mcastAddr, ipArr, emit)

	return func() {
		cancel()
		conn.Close()
	}
}

func runMDNSResponder(ctx context.Context, conn *net.UDPConn, mcast *net.UDPAddr, ip [4]byte, emit activity.Emitter) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		handleMDNSQuery(conn, mcast, src, buf[:n], ip, emit)
	}
}

func handleMDNSQuery(conn *net.UDPConn, mcast *net.UDPAddr, src *net.UDPAddr, data []byte, ip [4]byte, emit activity.Emitter) {
	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		return
	}
	if msg.Header.Response {
		return
	}

	srcStr := src.String()

	for _, q := range msg.Questions {
		name := q.Name.String()

		if name != stableHostname+"." {
			continue
		}

		if q.Type != dnsmessage.TypeA {
			diagMsg := fmt.Sprintf("mDNS: query from %s: %s %s — not answered (only A is supported)", srcStr, name, q.Type)
			log.Printf("[phone-intake] %s", diagMsg)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-query-skipped", diagMsg,
				map[string]any{"src": srcStr, "name": name, "type": q.Type.String(), "class": q.Class.String()}))
			continue
		}

		log.Printf("[phone-intake] mDNS: A query for %s from %s", name, srcStr)

		resp, err := buildMDNSResponse(msg.Header.ID, ip)
		if err != nil {
			log.Printf("[phone-intake] mDNS: build response error: %v", err)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-response-error",
				fmt.Sprintf("mDNS: failed to build response for %s query from %s: %v", name, srcStr, err),
				map[string]any{"src": srcStr, "name": name, "error": err.Error()}))
			return
		}

		dest := mcast
		destKind := "multicast"
		if src.Port != mdnsPort {
			dest = src
			destKind = "unicast"
		}

		if _, writeErr := conn.WriteToUDP(resp, dest); writeErr != nil {
			log.Printf("[phone-intake] mDNS: send %s response error: %v", destKind, writeErr)
			emit.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "mdns-send-error",
				fmt.Sprintf("mDNS: failed to send %s response for %s to %s: %v", destKind, name, dest, writeErr),
				map[string]any{"src": srcStr, "dest": dest.String(), "destKind": destKind, "name": name, "error": writeErr.Error()}))
			return
		}

		log.Printf("[phone-intake] mDNS: responded to A query for %s from %s → %s (%s)", name, srcStr, dest, destKind)
		emit.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "mdns-responded",
			fmt.Sprintf("mDNS: answered A query for %s from %s → %s (%s)", name, srcStr, dest, destKind),
			map[string]any{"src": srcStr, "dest": dest.String(), "destKind": destKind, "name": name}))
	}
}

func buildMDNSResponse(id uint16, ip [4]byte) ([]byte, error) {
	name, err := dnsmessage.NewName(stableHostname + ".")
	if err != nil {
		return nil, err
	}
	msg := dnsmessage.Message{
		Header: dnsmessage.Header{
			ID:            id,
			Response:      true,
			Authoritative: true,
		},
		Answers: []dnsmessage.Resource{
			{
				Header: dnsmessage.ResourceHeader{
					Name:  name,
					Type:  dnsmessage.TypeA,
					Class: dnsmessage.ClassINET,
					TTL:   mdnsAnswerTTL,
				},
				Body: &dnsmessage.AResource{A: ip},
			},
		},
	}
	return msg.Pack()
}

func sendMDNSAnnouncement(conn *net.UDPConn, mcast *net.UDPAddr, ip [4]byte) {
	resp, err := buildMDNSResponse(0, ip)
	if err != nil {
		return
	}
	_, _ = conn.WriteToUDP(resp, mcast)
}
