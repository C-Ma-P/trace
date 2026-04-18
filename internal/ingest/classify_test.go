package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
)

func TestClassifyFile(t *testing.T) {
	cases := []struct {
		name string
		file string
		want domain.AssetType
	}{
		{name: "symbol", file: "part.kicad_sym", want: domain.AssetTypeSymbol},
		{name: "footprint", file: "part.kicad_mod", want: domain.AssetTypeFootprint},
		{name: "step", file: "part.step", want: domain.AssetType3DModel},
		{name: "pdf", file: "part.pdf", want: domain.AssetTypeDatasheet},
		{name: "unsupported", file: "part.txt", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyFile(tc.file); got != tc.want {
				t.Fatalf("classifyFile(%q) = %q, want %q", tc.file, got, tc.want)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		got := ValidatePath(filepath.Join(t.TempDir(), "nope"))
		if got.Valid {
			t.Fatalf("expected invalid for missing path")
		}
		if got.PathKind != PathKindMissing {
			t.Fatalf("expected missing path kind, got %q", got.PathKind)
		}
	})

	t.Run("supported file", func(t *testing.T) {
		root := t.TempDir()
		f := filepath.Join(root, "asset.pdf")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("write test file: %v", err)
		}
		got := ValidatePath(f)
		if !got.Valid || got.PathKind != PathKindFile {
			t.Fatalf("expected valid file path, got %+v", got)
		}
	})

	t.Run("unsupported file", func(t *testing.T) {
		root := t.TempDir()
		f := filepath.Join(root, "asset.txt")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("write test file: %v", err)
		}
		got := ValidatePath(f)
		if got.Valid {
			t.Fatalf("expected unsupported file to be invalid")
		}
		if got.PathKind != PathKindFile {
			t.Fatalf("expected file path kind, got %q", got.PathKind)
		}
	})

	t.Run("directory with supported content", func(t *testing.T) {
		root := t.TempDir()
		f := filepath.Join(root, "lib.pretty", "part.kicad_mod")
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		got := ValidatePath(root)
		if !got.Valid || got.PathKind != PathKindDir {
			t.Fatalf("expected valid dir path, got %+v", got)
		}
	})

	t.Run("directory without supported content", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		got := ValidatePath(root)
		if got.Valid {
			t.Fatalf("expected invalid dir without supported files")
		}
		if got.PathKind != PathKindDir {
			t.Fatalf("expected dir path kind, got %q", got.PathKind)
		}
	})
}
