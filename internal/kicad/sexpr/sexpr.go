package sexpr

import (
	"fmt"
	"strings"
)

type W struct {
	b     strings.Builder
	depth int
}

func New() *W { return &W{} }

func (w *W) Line(content string) *W {
	w.writeIndent()
	w.b.WriteString(content)
	w.b.WriteByte('\n')
	return w
}

func (w *W) Enter(tag string, args ...string) *W {
	w.writeIndent()
	w.b.WriteByte('(')
	w.b.WriteString(tag)
	for _, a := range args {
		w.b.WriteByte(' ')
		w.b.WriteString(a)
	}
	w.b.WriteByte('\n')
	w.depth++
	return w
}

func (w *W) Leave() *W {
	w.depth--
	w.writeIndent()
	w.b.WriteString(")\n")
	return w
}

func (w *W) Raw(block string) *W {
	prefix := strings.Repeat("  ", w.depth)
	for _, line := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
		if line == "" {
			w.b.WriteByte('\n')
			continue
		}
		w.b.WriteString(prefix)
		w.b.WriteString(line)
		w.b.WriteByte('\n')
	}
	return w
}

func (w *W) String() string { return w.b.String() }

func (w *W) Bytes() []byte { return []byte(w.b.String()) }

func (w *W) writeIndent() {
	for i := 0; i < w.depth; i++ {
		w.b.WriteString("  ")
	}
}

func Q(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func FmtNum(v float64) string {
	s := fmt.Sprintf("%.6f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

type tokKind byte

const (
	tokOpen  tokKind = '('
	tokClose tokKind = ')'
	tokStr   tokKind = 's'
	tokAtom  tokKind = 'a'
)

type Tok struct {
	Kind tokKind
	Val  string
}

func Scan(src []byte) ([]Tok, error) {
	var out []Tok
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == '(':
			out = append(out, Tok{Kind: tokOpen})
			i++
		case c == ')':
			out = append(out, Tok{Kind: tokClose})
			i++
		case c == '"':
			s, n, err := readStr(src, i)
			if err != nil {
				return nil, err
			}
			out = append(out, Tok{Kind: tokStr, Val: s})
			i += n
		case c == ';':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case isWS(c):
			i++
		default:
			s, n := readAtom(src, i)
			out = append(out, Tok{Kind: tokAtom, Val: s})
			i += n
		}
	}
	return out, nil
}

func readStr(src []byte, start int) (string, int, error) {
	var sb strings.Builder
	i := start + 1
	for i < len(src) {
		c := src[i]
		if c == '"' {
			return sb.String(), i - start + 1, nil
		}
		if c == '\\' && i+1 < len(src) {
			i++
			switch src[i] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(src[i])
			}
		} else {
			sb.WriteByte(c)
		}
		i++
	}
	return "", 0, fmt.Errorf("unterminated string at offset %d", start)
}

func readAtom(src []byte, start int) (string, int) {
	i := start
	for i < len(src) && !isWS(src[i]) && src[i] != '(' && src[i] != ')' && src[i] != '"' && src[i] != ';' {
		i++
	}
	return string(src[start:i]), i - start
}

func isWS(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func ExtractSymbolBlocks(src []byte) ([][]Tok, error) {
	toks, err := Scan(src)
	if err != nil {
		return nil, err
	}
	if len(toks) < 2 || toks[0].Kind != tokOpen || toks[1].Kind != tokAtom || toks[1].Val != "kicad_symbol_lib" {
		return nil, fmt.Errorf("expected (kicad_symbol_lib …) wrapper; got %v", firstAtomVal(toks))
	}

	var result [][]Tok
	i := 2
	depth := 1
	for i < len(toks) && depth > 0 {
		t := toks[i]
		if t.Kind == tokOpen {
			if depth == 1 &&
				i+1 < len(toks) &&
				toks[i+1].Kind == tokAtom &&
				toks[i+1].Val == "symbol" {
				end := matchingClose(toks, i)
				if end < 0 {
					return nil, fmt.Errorf("unmatched ( for symbol block at token index %d", i)
				}
				result = append(result, toks[i:end+1])
				i = end + 1
				continue
			}
			depth++
		} else if t.Kind == tokClose {
			depth--
		}
		i++
	}
	return result, nil
}

func matchingClose(toks []Tok, start int) int {
	d := 0
	for i := start; i < len(toks); i++ {
		if toks[i].Kind == tokOpen {
			d++
		} else if toks[i].Kind == tokClose {
			d--
			if d == 0 {
				return i
			}
		}
	}
	return -1
}

func BlockName(toks []Tok) string {
	if len(toks) < 3 {
		return ""
	}
	if toks[0].Kind != tokOpen || toks[1].Kind != tokAtom || toks[1].Val != "symbol" {
		return ""
	}
	if toks[2].Kind != tokStr {
		return ""
	}
	return toks[2].Val
}

func RenameSymbolBlock(toks []Tok, oldName, newName string) []Tok {
	// KiCad requires sub-symbol names to use the unprefixed symbol name.
	// When newName is "lib:Name", sub-symbols must be "Name_0_1", not
	// "lib:Name_0_1".
	subPrefix := newName
	if idx := strings.LastIndex(newName, ":"); idx >= 0 {
		subPrefix = newName[idx+1:]
	}

	result := make([]Tok, len(toks))
	copy(result, toks)
	for i, t := range result {
		if t.Kind != tokStr {
			continue
		}
		if t.Val == oldName {
			result[i].Val = newName
		} else if isSubSymbolName(t.Val, oldName) {
			suffix := t.Val[len(oldName):]
			result[i].Val = subPrefix + suffix
		}
	}
	return result
}

func isSubSymbolName(name, parent string) bool {
	if !strings.HasPrefix(name, parent+"_") {
		return false
	}
	suffix := name[len(parent)+1:]
	parts := strings.SplitN(suffix, "_", 2)
	return len(parts) == 2 && isDigits(parts[0]) && isDigits(parts[1])
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func PinNumbers(toks []Tok) []string {
	var pins []string
	seen := make(map[string]bool)

	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Kind == tokOpen &&
			toks[i+1].Kind == tokAtom &&
			toks[i+1].Val == "number" &&
			toks[i+2].Kind == tokStr {
			num := toks[i+2].Val
			if !seen[num] {
				seen[num] = true
				pins = append(pins, num)
			}
		}
	}
	return pins
}

func SerializeTokens(toks []Tok) string {
	var b strings.Builder
	depth := 0

	for i, t := range toks {
		switch t.Kind {
		case tokOpen:
			if depth > 0 && i > 0 && toks[i-1].Kind != tokOpen {
				b.WriteByte('\n')
				writeSpaces(&b, depth)
			}
			b.WriteByte('(')
			depth++
		case tokClose:
			depth--
			b.WriteByte(')')
		case tokAtom:
			if i > 0 && toks[i-1].Kind != tokOpen {
				b.WriteByte(' ')
			}
			b.WriteString(t.Val)
		case tokStr:
			if i > 0 && toks[i-1].Kind != tokOpen {
				b.WriteByte(' ')
			}
			b.WriteByte('"')
			b.WriteString(escapeStr(t.Val))
			b.WriteByte('"')
		}
	}
	return b.String()
}

func writeSpaces(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteString("  ")
	}
}

func escapeStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func firstAtomVal(toks []Tok) string {
	for _, t := range toks {
		if t.Kind == tokAtom {
			return t.Val
		}
	}
	return "(empty)"
}
