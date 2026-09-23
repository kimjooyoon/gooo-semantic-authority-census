package census

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSourceRejectsTruncatedActivityDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.gooo")
	if err := os.WriteFile(path, []byte("activity CLASSIFY_UNKNOWN\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := parseSource(path)
	if err == nil || !strings.Contains(err.Error(), "malformed activity declaration at line 1") {
		t.Fatalf("parseSource() error = %v, want malformed declaration with line number", err)
	}
}

func TestParseGeneratedRejectsTruncatedBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "generated.go.txt")
	if err := os.WriteFile(path, []byte("// gooo-binding CLASSIFY_UNKNOWN\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := parseGenerated(path)
	if err == nil || !strings.Contains(err.Error(), "malformed // gooo-binding declaration at line 1") {
		t.Fatalf("parseGenerated() error = %v, want malformed declaration with line number", err)
	}
}
