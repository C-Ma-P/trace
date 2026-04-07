package phoneintake

import "strings"

func parseLcscBarcode(raw string) (partID, qty string) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	for _, pair := range strings.Split(s, ",") {
		idx := strings.IndexByte(pair, ':')
		if idx < 0 {
			continue
		}
		k := strings.TrimSpace(pair[:idx])
		v := strings.TrimSpace(pair[idx+1:])
		switch k {
		case "pc":
			partID = v
		case "qty":
			qty = v
		}
	}
	return
}

func parseMouserBarcode(raw string) (partNumber, qty string) {
	payload := raw
	if strings.HasPrefix(payload, "[)>") {
		if idx := strings.Index(payload, "06"); idx >= 0 {
			payload = payload[idx+2:]
		}
	}

	var segments []string
	if strings.ContainsAny(payload, "\x1d\x1e") {
		for _, seg := range strings.FieldsFunc(payload, func(r rune) bool {
			return r == '\x1d' || r == '\x1e' || r == '\x04'
		}) {
			if seg != "" {
				segments = append(segments, seg)
			}
		}
	} else {
		segments = splitMouserFallback(payload)
	}

	for _, seg := range segments {
		switch {
		case strings.HasPrefix(seg, "21P"):
			partNumber = seg[3:]
		case strings.HasPrefix(seg, "1P") && partNumber == "":
			partNumber = seg[2:]
		case strings.HasPrefix(seg, "Q") && qty == "":
			qty = seg[1:]
		}
	}
	return
}

func splitMouserFallback(payload string) []string {
	prefixes := []string{"21P", "11K", "1P", "1V", "LT", "Q", "K"}
	var segments []string
	i := 0
	for i < len(payload) {
		matched := false
		for _, pfx := range prefixes {
			if strings.HasPrefix(payload[i:], pfx) {
				end := len(payload)
			outer:
				for j := i + len(pfx); j < len(payload); j++ {
					for _, p2 := range prefixes {
						if strings.HasPrefix(payload[j:], p2) {
							end = j
							break outer
						}
					}
				}
				segments = append(segments, payload[i:end])
				i = end
				matched = true
				break
			}
		}
		if !matched {
			i++
		}
	}
	return segments
}
