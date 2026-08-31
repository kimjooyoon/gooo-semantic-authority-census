package census

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDecisionPrecedenceAndAuthority(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		missing bool
		want    string
	}{
		{name: "generated", kind: "GENERATED_FROM_GOOO", want: "CLOSED"},
		{name: "missing", kind: "GENERATED_FROM_GOOO", missing: true, want: "UNKNOWN"},
		{name: "handwritten", kind: "HANDWRITTEN_GO", want: "REFUTED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			policyPath := filepath.Join(root, "policy.json")
			manifestPath := filepath.Join(root, "manifest.json")
			sourcePath := filepath.Join(root, "source.gooo")
			irPath := filepath.Join(root, "ir.json")
			generatedPath := filepath.Join(root, "generated.txt")
			writeJSON(t, policyPath, testPolicy())
			if err := os.WriteFile(sourcePath, []byte("activity A semantic=one\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			writeJSON(t, irPath, irDocument{Activities: []irActivity{{ID: "A", Semantic: "semantic=one"}}})
			if !tt.missing {
				if err := os.WriteFile(generatedPath, []byte("// gooo-binding A semantic=one\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			writeJSON(t, manifestPath, Manifest{
				Schema:           "gooo/semantic-authority-census-manifest/v1",
				ScenarioID:       tt.name,
				ExpectedDecision: tt.want,
				Freshness:        "CURRENT",
				Obligations: []Obligation{{
					ID: "A", SourcePath: "source.gooo", IRPath: "ir.json",
					GeneratedPath: "generated.txt", ImplementationKind: tt.kind,
				}},
			})
			report, err := Evaluate(policyPath, manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			if report.Decision != tt.want {
				t.Fatalf("decision=%s want=%s", report.Decision, tt.want)
			}
			if report.Metrics.ReplayComparisons != 1 || report.Metrics.ReplayMismatches != 0 {
				t.Fatalf("replay=%d/%d", report.Metrics.ReplayComparisons, report.Metrics.ReplayMismatches)
			}
		})
	}
}

func testPolicy() Policy {
	cells := []PolicyCell{
		{ID: "LOAD_POLICY"}, {ID: "BIND_SOURCE"}, {ID: "BIND_SEMANTIC_IR"}, {ID: "BIND_GENERATED_GO"},
		{ID: "VERIFY_SOURCE_IR"}, {ID: "VERIFY_IR_GENERATED"}, {ID: "CLASSIFY_IMPLEMENTATION_AUTHORITY"},
		{ID: "VERIFY_INPUT_FRESHNESS"}, {ID: "PRESERVE_UNKNOWN_FRONTIER"}, {ID: "PRESERVE_REFUTATION"},
		{ID: "VERIFY_DETERMINISTIC_REPLAY"}, {ID: "PUBLISH_HUMAN_REPORT"},
	}
	return Policy{
		ID:            "test",
		Precedence:    []string{"REFUTED", "UNKNOWN", "CLOSED"},
		UnknownFields: []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"},
		Cells:         cells,
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
