package mdns

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

func classifyQuery(src *net.UDPAddr, questions []dns.Question) (isLegacy, quBitSet bool) {
	isLegacy = src.Port != mcastPort
	const quBit = 0x8000
	for _, q := range questions {
		if q.Qclass&quBit != 0 {
			quBitSet = true
			break
		}
	}
	return
}

func answersForQuestion(hostname string, q dns.Question, aRec dns.RR, ip6Addrs []net.IP) []dns.RR {
	name := q.Name
	if len(name) > 0 && name[len(name)-1] == '.' {
		name = name[:len(name)-1]
	}
	if name != hostname {
		return nil
	}
	switch q.Qtype {
	case dns.TypeA:
		return []dns.RR{aRec}
	case dns.TypeANY:
		ans := []dns.RR{aRec}
		for _, ip6 := range ip6Addrs {
			ans = append(ans, buildAAAARecord(hostname, ip6))
		}
		return ans
	case dns.TypeAAAA:
		if len(ip6Addrs) > 0 {
			ans := make([]dns.RR, len(ip6Addrs))
			for i, ip6 := range ip6Addrs {
				ans[i] = buildAAAARecord(hostname, ip6)
			}
			return ans
		}
		return []dns.RR{buildNSECRecord(hostname)}
	case dns.TypeHTTPS:
		return []dns.RR{buildNSECNoHTTPSRecord(hostname)}
	}
	return nil
}

func sendAnnounce(conn *net.UDPConn, aRec dns.RR) error {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Answer = []dns.RR{aRec}
	packed, err := msg.Pack()
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(packed, mcastGroupAddr)
	return err
}

func respond(ctx context.Context, hostname string, conn *net.UDPConn, ip4 net.IP, ip6Addrs []net.IP, hooks Hooks) {
	buf := make([]byte, 65535)
	aRec := buildARecord(hostname, ip4)

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
			return
		}

		var query dns.Msg
		if err := query.Unpack(buf[:n]); err != nil {
			continue
		}
		if query.Response {
			continue
		}

		isLegacy, quBitSet := classifyQuery(src, query.Question)

		var answers []dns.RR
		for _, q := range query.Question {
			answers = append(answers, answersForQuestion(hostname, q, aRec, ip6Addrs)...)
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
		if isLegacy {
			resp.Answer = legacyAnswers(answers)
		} else {
			resp.Answer = answers
			resp.Question = nil
		}

		packed, err := resp.Pack()
		if err != nil {
			continue
		}

		switch {
		case quBitSet:
			log.Printf("[phone-intake] mDNS: QU unicast reply to %s (%d answer(s))", src, len(resp.Answer))
			if _, err := conn.WriteToUDP(packed, src); err != nil {
				hooks.warn("mdns-write-error",
					fmt.Sprintf("mDNS: unicast write error to %s: %v", src, err),
					map[string]any{"dst": src.String(), "error": err.Error()})
			}
		case isLegacy:
			log.Printf("[phone-intake] mDNS: legacy-unicast reply to %s (%d answer(s))", src, len(resp.Answer))
			if _, err := conn.WriteToUDP(packed, src); err != nil {
				hooks.warn("mdns-write-error",
					fmt.Sprintf("mDNS: legacy unicast write error to %s: %v", src, err),
					map[string]any{"dst": src.String(), "error": err.Error()})
			}
		default:
			if _, err := conn.WriteToUDP(packed, mcastGroupAddr); err != nil {
				hooks.warn("mdns-write-error",
					fmt.Sprintf("mDNS: multicast write error: %v", err),
					map[string]any{"error": err.Error()})
			}
		}
	}
}
