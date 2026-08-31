package census

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

func Evaluate(policyPath, manifestPath string) (Report, error) {
	first, err := evaluateOnce(policyPath, manifestPath)
	if err != nil {
		return Report{}, err
	}
	second, err := evaluateOnce(policyPath, manifestPath)
	if err != nil {
		return Report{}, err
	}
	first.Metrics.ReplayComparisons = 1
	second.Metrics.ReplayComparisons = 1
	if !reflect.DeepEqual(replayProjection(first), replayProjection(second)) {
		first.Metrics.ReplayMismatches = 1
		appendRefutation(&first, "VERIFY_DETERMINISTIC_REPLAY", Refutation{
			Stage: "REPLAY", Step: "COMPARE_CANONICAL_REPORT", Reason: "DETERMINISTIC_REPLAY_MISMATCH",
			Counterexample: "two observations of the same immutable inputs differ",
		})
	} else {
		setCell(&first, "VERIFY_DETERMINISTIC_REPLAY", "CLOSED", "")
	}
	first.Decision = reduce(first)
	return first, nil
}

func evaluateOnce(policyPath, manifestPath string) (Report, error) {
	var p Policy
	if err := readJSON(policyPath, &p); err != nil {
		return Report{}, err
	}
	if err := validatePolicy(p); err != nil {
		return Report{}, err
	}
	var manifest Manifest
	if err := readJSON(manifestPath, &manifest); err != nil {
		return Report{}, err
	}
	if len(manifest.Obligations) == 0 {
		return Report{}, errors.New("manifest has no obligations")
	}
	report := Report{
		Schema:           "gooo/semantic-authority-census-report/v1",
		ScenarioID:       manifest.ScenarioID,
		ExpectedDecision: manifest.ExpectedDecision,
		PolicyID:         p.ID,
		Score:            "NOT_COMBINED",
		Unknowns:         []Unknown{},
		Refutations:      []Refutation{},
		NextFrontier:     []string{},
		FileDigests:      map[string]string{},
		Metrics: Metrics{
			ObligationsTotal:        len(manifest.Obligations),
			RepositoryWrites:        manifest.Authority.RepositoryWrites,
			LocalTestExecutions:     manifest.Authority.LocalTestExecutions,
			InfrastructureMutations: manifest.Authority.InfrastructureMutations,
			ProviderInstallAttempts: manifest.Authority.ProviderInstallAttempts,
			NetworkMutationAttempts: manifest.Authority.NetworkMutationAttempts,
		},
	}
	for _, c := range p.Cells {
		report.Cells = append(report.Cells, CellResult{
			ID: c.ID, Activity: c.Activity, Proof: c.Proof, Indicator: c.Indicator, State: "CLOSED",
		})
	}
	sourceFiles := map[string]bool{}
	irFiles := map[string]bool{}
	generatedFiles := map[string]bool{}
	unknownIDs := map[string]bool{}
	refutedIDs := map[string]bool{}

	if manifest.Freshness != "CURRENT" {
		appendUnknown(&report, "VERIFY_INPUT_FRESHNESS", Unknown{
			Stage: "INPUT_FRESHNESS", Step: "VERIFY_INPUT_DIGEST", Reason: "INPUT_NOT_CURRENT",
			UnknownClass: "STALE", NextOperation: "PIN_CURRENT_IMMUTABLE_INPUT", BlockedBy: []string{"INPUT_RELEASE_DIGEST"},
		})
	}
	if manifest.Authority.RepositoryWrites != 0 || manifest.Authority.LocalTestExecutions != 0 ||
		manifest.Authority.InfrastructureMutations != 0 || manifest.Authority.ProviderInstallAttempts != 0 ||
		manifest.Authority.NetworkMutationAttempts != 0 {
		appendRefutation(&report, "CLASSIFY_IMPLEMENTATION_AUTHORITY", Refutation{
			Stage: "AUTHORITY", Step: "CHECK_OBSERVED_AUTHORITY", Reason: "AUTHORITY_ESCALATION",
			Counterexample: "one or more forbidden authority counters are non-zero",
		})
	}

	base := filepath.Dir(manifestPath)
	for _, obligation := range manifest.Obligations {
		obligationUnknown := false
		obligationRefuted := false
		sourcePath := filepath.Clean(filepath.Join(base, obligation.SourcePath))
		irPath := filepath.Clean(filepath.Join(base, obligation.IRPath))
		generatedPath := filepath.Clean(filepath.Join(base, obligation.GeneratedPath))

		source, sourceErr := parseSource(sourcePath)
		if sourceErr != nil {
			obligationUnknown = true
			appendUnknown(&report, "BIND_SOURCE", missingUnknown("SOURCE", "READ_GOOO_SOURCE", "SOURCE_FILE_MISSING", "IMPORT_GOOO_SOURCE", obligation.ID))
		} else {
			sourceFiles[obligation.SourcePath] = true
			recordDigest(&report, obligation.SourcePath, sourcePath)
		}
		ir, irErr := parseIR(irPath)
		if irErr != nil {
			obligationUnknown = true
			appendUnknown(&report, "BIND_SEMANTIC_IR", missingUnknown("SEMANTIC_IR", "READ_SEMANTIC_IR", "IR_FILE_MISSING_OR_MALFORMED", "REGENERATE_SEMANTIC_IR", obligation.ID))
		} else {
			irFiles[obligation.IRPath] = true
			recordDigest(&report, obligation.IRPath, irPath)
		}
		generated, generatedErr := parseGenerated(generatedPath)
		if generatedErr != nil {
			obligationUnknown = true
			appendUnknown(&report, "BIND_GENERATED_GO", missingUnknown("GENERATED_BINDING", "READ_GENERATED_GO", "GENERATED_FILE_MISSING", "GENERATE_FROM_GOOO", obligation.ID))
		} else {
			generatedFiles[obligation.GeneratedPath] = true
			recordDigest(&report, obligation.GeneratedPath, generatedPath)
		}

		var sourceSemantic, irSemantic, generatedSemantic string
		if sourceErr == nil {
			sourceSemantic = source.values[obligation.ID]
			if source.ambiguous[obligation.ID] {
				obligationUnknown = true
				appendUnknown(&report, "BIND_SOURCE", Unknown{
					Stage: "SOURCE_BINDING", Step: "SELECT_ACTIVITY", Reason: "SOURCE_ACTIVITY_AMBIGUOUS",
					UnknownClass: "AMBIGUOUS", NextOperation: "DISAMBIGUATE_GOOO_ACTIVITY",
					BlockedBy: []string{"SOURCE_ACTIVITY:" + obligation.ID}, ObligationID: obligation.ID,
				})
			} else if sourceSemantic == "" {
				obligationUnknown = true
				appendUnknown(&report, "BIND_SOURCE", missingUnknown("SOURCE_BINDING", "SELECT_ACTIVITY", "SOURCE_ACTIVITY_MISSING", "DECLARE_GOOO_ACTIVITY", obligation.ID))
			}
		}
		if irErr == nil {
			irSemantic = ir.values[obligation.ID]
			if ir.ambiguous[obligation.ID] || irSemantic == "" {
				obligationUnknown = true
				appendUnknown(&report, "BIND_SEMANTIC_IR", Unknown{
					Stage: "IR_BINDING", Step: "SELECT_IR_ACTIVITY", Reason: "IR_ACTIVITY_MISSING_OR_AMBIGUOUS",
					UnknownClass: "AMBIGUOUS", NextOperation: "REGENERATE_UNAMBIGUOUS_IR",
					BlockedBy: []string{"IR_ACTIVITY:" + obligation.ID}, ObligationID: obligation.ID,
				})
			}
		}
		if generatedErr == nil {
			generatedSemantic = generated.values[obligation.ID]
			if generated.ambiguous[obligation.ID] || generatedSemantic == "" {
				obligationUnknown = true
				appendUnknown(&report, "BIND_GENERATED_GO", Unknown{
					Stage: "GENERATED_BINDING", Step: "SELECT_GENERATED_ACTIVITY", Reason: "GENERATED_ACTIVITY_MISSING_OR_AMBIGUOUS",
					UnknownClass: "AMBIGUOUS", NextOperation: "REGENERATE_GO_BINDING",
					BlockedBy: []string{"GENERATED_ACTIVITY:" + obligation.ID}, ObligationID: obligation.ID,
				})
			}
		}

		if obligation.ImplementationKind == "HANDWRITTEN_GO" {
			report.Metrics.HandwrittenGo++
			obligationRefuted = true
			appendRefutation(&report, "CLASSIFY_IMPLEMENTATION_AUTHORITY", Refutation{
				Stage: "AUTHORITY", Step: "CLASSIFY_IMPLEMENTATION", Reason: "HANDWRITTEN_SEMANTIC_AUTHORITY",
				ObligationID: obligation.ID, Counterexample: obligation.GeneratedPath + " is declared as handwritten Go",
			})
		} else if obligation.ImplementationKind != "GENERATED_FROM_GOOO" {
			obligationUnknown = true
			appendUnknown(&report, "CLASSIFY_IMPLEMENTATION_AUTHORITY", Unknown{
				Stage: "AUTHORITY", Step: "CLASSIFY_IMPLEMENTATION", Reason: "IMPLEMENTATION_KIND_UNKNOWN",
				UnknownClass: "UNBOUNDED", NextOperation: "DECLARE_IMPLEMENTATION_AUTHORITY",
				BlockedBy: []string{}, ObligationID: obligation.ID,
			})
		}

		if !obligationUnknown {
			if sourceSemantic != irSemantic {
				obligationRefuted = true
				appendRefutation(&report, "VERIFY_SOURCE_IR", Refutation{
					Stage: "SEMANTIC_BINDING", Step: "COMPARE_SOURCE_IR", Reason: "SOURCE_IR_CONTRADICTION",
					ObligationID: obligation.ID, Counterexample: fmt.Sprintf("source=%q ir=%q", sourceSemantic, irSemantic),
				})
			} else {
				report.Metrics.SemanticRelations++
			}
			if irSemantic != generatedSemantic {
				obligationRefuted = true
				appendRefutation(&report, "VERIFY_IR_GENERATED", Refutation{
					Stage: "SEMANTIC_BINDING", Step: "COMPARE_IR_GENERATED", Reason: "IR_GENERATED_CONTRADICTION",
					ObligationID: obligation.ID, Counterexample: fmt.Sprintf("ir=%q generated=%q", irSemantic, generatedSemantic),
				})
			} else {
				report.Metrics.SemanticRelations++
			}
		}
		if obligationUnknown {
			unknownIDs[obligation.ID] = true
		}
		if obligationRefuted {
			refutedIDs[obligation.ID] = true
		}
		if !obligationUnknown && !obligationRefuted && obligation.ImplementationKind == "GENERATED_FROM_GOOO" {
			report.Metrics.GeneratedBound++
		}
		if obligationUnknown || obligationRefuted {
			report.NextFrontier = append(report.NextFrontier, obligation.ID)
		}
	}
	report.Metrics.SourceFilesObserved = len(sourceFiles)
	report.Metrics.IRFilesObserved = len(irFiles)
	report.Metrics.GeneratedFilesObserved = len(generatedFiles)
	report.Metrics.UnknownObligations = len(unknownIDs)
	report.Metrics.RefutedObligations = len(refutedIDs)
	sort.Strings(report.NextFrontier)
	report.NextFrontier = unique(report.NextFrontier)
	report.Decision = reduce(report)
	return report, nil
}

func validatePolicy(p Policy) error {
	if len(p.Cells) != 12 {
		return fmt.Errorf("policy denominator is %d, expected 12", len(p.Cells))
	}
	if !reflect.DeepEqual(p.Precedence, []string{"REFUTED", "UNKNOWN", "CLOSED"}) {
		return errors.New("policy precedence is not REFUTED > UNKNOWN > CLOSED")
	}
	if !reflect.DeepEqual(p.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) {
		return errors.New("policy UNKNOWN schema is not the fixed six fields")
	}
	return nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func missingUnknown(stage, step, reason, next, obligationID string) Unknown {
	return Unknown{
		Stage: stage, Step: step, Reason: reason, UnknownClass: "DIRECT_MISSING",
		NextOperation: next, BlockedBy: []string{}, ObligationID: obligationID,
	}
}

func appendUnknown(report *Report, cellID string, value Unknown) {
	report.Unknowns = append(report.Unknowns, value)
	setCell(report, cellID, "UNKNOWN", value.Reason)
	setCell(report, "PRESERVE_UNKNOWN_FRONTIER", "UNKNOWN", value.Reason)
}

func appendRefutation(report *Report, cellID string, value Refutation) {
	report.Refutations = append(report.Refutations, value)
	setCell(report, cellID, "REFUTED", value.Reason)
	setCell(report, "PRESERVE_REFUTATION", "REFUTED", value.Reason)
}

func setCell(report *Report, id, state, reason string) {
	rank := map[string]int{"CLOSED": 0, "UNKNOWN": 1, "REFUTED": 2}
	for i := range report.Cells {
		if report.Cells[i].ID == id && rank[state] >= rank[report.Cells[i].State] {
			report.Cells[i].State = state
			report.Cells[i].Reason = reason
			return
		}
	}
}

func reduce(report Report) string {
	if len(report.Refutations) > 0 {
		return "REFUTED"
	}
	if len(report.Unknowns) > 0 {
		return "UNKNOWN"
	}
	return "CLOSED"
}

func recordDigest(report *Report, logical, actual string) {
	if _, exists := report.FileDigests[logical]; exists {
		return
	}
	if digest, err := fileDigest(actual); err == nil {
		report.FileDigests[logical] = digest
	}
}

func replayProjection(report Report) any {
	return struct {
		Decision     string
		Cells        []CellResult
		Unknowns     []Unknown
		Refutations  []Refutation
		NextFrontier []string
		FileDigests  map[string]string
		Metrics      Metrics
	}{
		Decision:     report.Decision,
		Cells:        report.Cells,
		Unknowns:     report.Unknowns,
		Refutations:  report.Refutations,
		NextFrontier: report.NextFrontier,
		FileDigests:  report.FileDigests,
		Metrics:      report.Metrics,
	}
}

func unique(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := []string{values[0]}
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
