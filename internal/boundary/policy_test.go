package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePolicyRejectsDuplicateSingletonDirective(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.gooo")
	source := "boundary_policy first\nboundary_policy second\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ParsePolicy(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate singleton boundary directive: boundary_policy") {
		t.Fatalf("ParsePolicy() error = %v, want duplicate singleton directive", err)
	}
}
