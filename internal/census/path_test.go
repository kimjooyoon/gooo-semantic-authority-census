package census

import (
	"path/filepath"
	"testing"
)

func TestBoundedManifestPathRejectsAbsoluteAndTraversalPaths(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "cases")
	if _, err := boundedManifestPath(root, base, "../common/source.gooo"); err != nil {
		t.Fatalf("boundedManifestPath rejected an in-root sibling path: %v", err)
	}
	for _, path := range []string{"../../outside.json", "/tmp/outside.json"} {
		if _, err := boundedManifestPath(root, base, path); err == nil {
			t.Fatalf("boundedManifestPath accepted %q", path)
		}
	}
}
