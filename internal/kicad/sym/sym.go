package sym

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/C-Ma-P/trace/internal/kicad/sexpr"
)

const libVersion = "20220914"
const libGenerator = "trace-export"

type entry struct {
	localKey string
	origName string
	toks     []sexpr.Tok
	pinNums  []string
}

type Library struct {
	entries []entry
	byKey   map[string]bool
}

func New() *Library {
	return &Library{byKey: make(map[string]bool)}
}

func (l *Library) Has(localKey string) bool {
	return l.byKey[localKey]
}

func (l *Library) AddFromFile(localKey, path, nameHint string) error {
	if l.byKey[localKey] {
		return nil
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read symbol file %q: %w", path, err)
	}

	blocks, err := sexpr.ExtractSymbolBlocks(src)
	if err != nil {
		return fmt.Errorf("parse symbol file %q: %w", path, err)
	}
	if len(blocks) == 0 {
		return fmt.Errorf("no symbols found in %q", path)
	}

	block := blocks[0]
	if nameHint != "" {
		for _, b := range blocks {
			if name := sexpr.BlockName(b); strings.Contains(name, nameHint) {
				block = b
				break
			}
		}
	}

	origName := sexpr.BlockName(block)
	if origName == "" {
		return fmt.Errorf("symbol block in %q has no name", path)
	}

	pinNums := sexpr.PinNumbers(block)

	l.entries = append(l.entries, entry{
		localKey: localKey,
		origName: origName,
		toks:     block,
		pinNums:  pinNums,
	})
	l.byKey[localKey] = true
	return nil
}

func (l *Library) LocalKeys() []string {
	keys := make([]string, len(l.entries))
	for i, e := range l.entries {
		keys[i] = e.localKey
	}
	return keys
}

func (l *Library) PinNumbers(localKey string) []string {
	for _, e := range l.entries {
		if e.localKey == localKey {
			return e.pinNums
		}
	}
	return nil
}

func (l *Library) WriteTo(w io.Writer) error {
	out := sexpr.New()
	out.Enter("kicad_symbol_lib")
	out.Line(fmt.Sprintf("(version %s)", libVersion))
	out.Line(fmt.Sprintf("(generator %s)", sexpr.Q(libGenerator)))

	for _, e := range l.entries {

		renamed := sexpr.RenameSymbolBlock(e.toks, e.origName, e.localKey)
		out.Raw(sexpr.SerializeTokens(renamed))
	}
	out.Leave()

	_, err := io.WriteString(w, out.String())
	return err
}

func (l *Library) WriteLibSymbolsTo(w io.Writer, libPrefix string) error {
	for _, e := range l.entries {
		qualifiedName := libPrefix + ":" + e.localKey
		renamed := sexpr.RenameSymbolBlock(e.toks, e.origName, qualifiedName)
		_, err := io.WriteString(w, sexpr.SerializeTokens(renamed)+"\n")
		if err != nil {
			return err
		}
	}
	return nil
}
