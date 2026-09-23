package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRejectsDuplicateSingletonDirectives(t *testing.T) {
	for _, test := range []struct {
		name string
		line string
	}{
		{name: "policy", line: "policy semantic-authority-census-v1"},
		{name: "precedence", line: "precedence REFUTED UNKNOWN CLOSED"},
		{name: "unknown fields", line: "unknown_fields stage step reason unknown_class next_operation blocked_by"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "policy.gooo")
			content := test.line + "\n" + test.line + "\n"
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := parse(path); err == nil {
				t.Fatalf("parse accepted duplicate %s directive", test.name)
			}
		})
	}
}
