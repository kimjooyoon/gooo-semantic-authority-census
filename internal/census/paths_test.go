package census

import "testing"

func TestContainedPathRejectsEscape(t *testing.T) {
	if _, err := containedPath("/tmp/input", "../outside.json"); err == nil {
		t.Fatal("expected path escaping input root to be rejected")
	}
}

func TestContainedPathAllowsNestedRelativePath(t *testing.T) {
	path, err := containedPath("/tmp/input", "nested/source.gooo")
	if err != nil || path != "/tmp/input/nested/source.gooo" {
		t.Fatalf("containedPath() = %q, %v", path, err)
	}
}
