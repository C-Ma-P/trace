package mdns

import (
	"net"

	"github.com/miekg/dns"
)

func buildARecord(hostname string, ip4 net.IP) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   hostname + ".",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET | 0x8000,
			Ttl:    hostTTL,
		},
		A: ip4,
	}
}

func buildAAAARecord(hostname string, ip6 net.IP) *dns.AAAA {
	return &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   hostname + ".",
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET | 0x8000,
			Ttl:    hostTTL,
		},
		AAAA: ip6,
	}
}

func buildNSECRecord(hostname string) *dns.NSEC {
	return &dns.NSEC{
		Hdr: dns.RR_Header{
			Name:   hostname + ".",
			Rrtype: dns.TypeNSEC,
			Class:  dns.ClassINET | 0x8000,
			Ttl:    hostTTL,
		},
		NextDomain: hostname + ".",
		TypeBitMap: []uint16{dns.TypeA},
	}
}

func buildNSECNoHTTPSRecord(hostname string) *dns.NSEC {
	return &dns.NSEC{
		Hdr: dns.RR_Header{
			Name:   hostname + ".",
			Rrtype: dns.TypeNSEC,
			Class:  dns.ClassINET | 0x8000,
			Ttl:    hostTTL,
		},
		NextDomain: hostname + ".",
		TypeBitMap: []uint16{dns.TypeA},
	}
}

func legacyAnswers(answers []dns.RR) []dns.RR {
	out := make([]dns.RR, 0, len(answers))
	for _, rr := range answers {
		tmp := (&dns.Msg{Answer: []dns.RR{rr}}).Copy()
		cp := tmp.Answer[0]
		hdr := cp.Header()
		hdr.Class &^= 0x8000
		if hdr.Ttl > legacyMaxTTL {
			hdr.Ttl = legacyMaxTTL
		}
		out = append(out, cp)
	}
	return out
}
