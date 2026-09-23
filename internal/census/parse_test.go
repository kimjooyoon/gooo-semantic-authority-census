package census

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseIRRejectsDuplicateJSONKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "semantic-ir.json")
	data := []byte(`{"activities":[{"id":"activity-a","id":"activity-b","semantic":"stable"}]}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseIR(path); err == nil {
		t.Fatal("semantic IR with duplicate JSON keys was accepted")
	}
}
