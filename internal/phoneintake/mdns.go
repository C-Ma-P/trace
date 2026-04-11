package phoneintake

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	mdnsMulticastAddr = "224.0.0.251"
	mdnsPort          = 5353
	mdnsAnswerTTL     = 120
)

func startMDNS(lanIP string, onWarn func(string)) func() {
	ip := net.ParseIP(lanIP)
	if ip == nil {
		onWarn("mDNS: no valid LAN IP — trace.local will not resolve, use IP URL as fallback")
		return func() {}
	}
	ip4 := ip.To4()
	if ip4 == nil {
		onWarn(fmt.Sprintf("mDNS: LAN IP %s is not IPv4 — trace.local A record will not be advertised", lanIP))
		return func() {}
	}
	var ipArr [4]byte
	copy(ipArr[:], ip4)

	mcastAddr := &net.UDPAddr{IP: net.ParseIP(mdnsMulticastAddr), Port: mdnsPort}

	conn, err := net.ListenMulticastUDP("udp4", nil, mcastAddr)
	if err != nil {
		onWarn(fmt.Sprintf("mDNS: bind failed (%v) — trace.local will not resolve; use LAN IP URL as fallback", err))
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		sendMDNSAnnouncement(conn, mcastAddr, ipArr)
	}()

	go runMDNSResponder(ctx, conn, mcastAddr, ipArr)

	return func() {
		cancel()
		conn.Close()
	}
}

func runMDNSResponder(ctx context.Context, conn *net.UDPConn, mcast *net.UDPAddr, ip [4]byte) {
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

		handleMDNSQuery(conn, mcast, src, buf[:n], ip)
	}
}

func handleMDNSQuery(conn *net.UDPConn, mcast *net.UDPAddr, src *net.UDPAddr, data []byte, ip [4]byte) {
	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		return
	}
	if msg.Header.Response {
		return
	}

	for _, q := range msg.Questions {
		if q.Type != dnsmessage.TypeA {
			continue
		}
		if q.Name.String() != stableHostname+"." {
			continue
		}
		resp, err := buildMDNSResponse(msg.Header.ID, ip)
		if err != nil {
			return
		}
		// RFC 6762 §5.1 one-shot: if the query came from a non-zero source port
		// (Android / one-shot) reply directly to the sender (unicast).
		// Standard Bonjour queries come from port 5353; reply to multicast.
		dest := mcast
		if src.Port != mdnsPort {
			dest = src
		}
		_, _ = conn.WriteToUDP(resp, dest)
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
