package census

import "testing"

func TestBoundedManifestPathRejectsAbsoluteAndTraversalPaths(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"../outside.json", "/tmp/outside.json"} {
		if _, err := boundedManifestPath(root, path); err == nil {
			t.Fatalf("boundedManifestPath accepted %q", path)
		}
	}
}
