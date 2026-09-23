package boundary

import "testing"

func TestAuthorityStateVocabularyIsFixed(t *testing.T) {
	states := AuthorityStates()
	if len(states) != 6 {
		t.Fatalf("authority states=%d want 6", len(states))
	}
	if !IsOwnedAuthorityState("GOOO_OWNED") || !IsOwnedAuthorityState("BOOTSTRAP_EXTERNAL") {
		t.Fatal("owner states were not preserved")
	}
	if IsOwnedAuthorityState(DecisionUnknown) || IsOwnedAuthorityState(DecisionRefuted) {
		t.Fatal("UNKNOWN and REFUTED must not be caller-asserted owner states")
	}
}

func TestAuthorityStateVocabularyCannotBeMutatedThroughSnapshot(t *testing.T) {
	states := AuthorityStates()
	states[0] = DecisionRefuted
	if AuthorityStates()[0] == DecisionRefuted {
		t.Fatal("authority vocabulary exposed mutable storage")
	}
}

func TestSemanticMarkerParsingDoesNotUseFileShape(t *testing.T) {
	text := "// boundary cell=EVALUATOR_RULES authority=HANDWRITTEN_RUNTIME semantic_role=evaluator_rules evidence_kind=HANDWRITTEN_GO"
	if authorityFromText(text) != "HANDWRITTEN_RUNTIME" {
		t.Fatalf("authority=%q", authorityFromText(text))
	}
	if authorityFromText("a .gooo file with 100 lines") != "" {
		t.Fatal("file shape unexpectedly supplied semantic authority")
	}
}

func TestUniqueSorted(t *testing.T) {
	got := uniqueSorted([]string{"B", "A", "B", "C"})
	want := []string{"A", "B", "C"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}
