package census

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLineBindingsPreservesSemanticSpacing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.gooo")
	const semantic = "semantic=one  with\tspacing"
	if err := os.WriteFile(path, []byte("activity A "+semantic+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := parseSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.values["A"]; got != semantic {
		t.Fatalf("semantic=%q, want %q", got, semantic)
	}
}
