package boundary

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Project(policyPath, manifestPath, outputDir, repositoryRoot string) (Report, error) {
	first, err := evaluateOnce(policyPath, manifestPath, repositoryRoot)
	if err != nil {
		return Report{}, err
	}
	second, err := evaluateOnce(policyPath, manifestPath, repositoryRoot)
	if err != nil {
		return Report{}, err
	}
	first.Replay = replayResult(first, second)
	first.Decision = reduce(first)
	if first.Replay.Mismatches > 0 {
		first.Refutations = append(first.Refutations, Refutation{
			Stage: "REPLAY", Step: "COMPARE_IMMUTABLE_PROJECTION", Reason: "DETERMINISTIC_REPLAY_MISMATCH",
			Counterexample: "the same policy, manifest, and evidence files produced different projections",
		})
		first.Decision = DecisionRefuted
	}
	if first.ExpectedDecision != "" && first.Decision != first.ExpectedDecision {
		return Report{}, fmt.Errorf("scenario %s decided %s, expected %s", first.ScenarioID, first.Decision, first.ExpectedDecision)
	}
	if err := writeReport(outputDir, first); err != nil {
		return Report{}, err
	}
	return first, nil
}

func evaluateOnce(policyPath, manifestPath, repositoryRoot string) (Report, error) {
	compiled, err := LoadCompiledPolicy(policyPath)
	if err != nil {
		return Report{}, err
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return Report{}, err
	}
	var manifest InputManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return Report{}, err
	}
	if manifest.Schema != InputSchema {
		return Report{}, fmt.Errorf("unexpected input schema %q", manifest.Schema)
	}
	if manifest.ScenarioID == "" {
		return Report{}, errors.New("scenario_id is required")
	}

	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return Report{}, err
	}
	manifestBase, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		return Report{}, err
	}
	policyDigest := compiled.SourceDigest
	report := Report{
		Schema: ReportSchema, ScenarioID: manifest.ScenarioID, ExpectedDecision: manifest.ExpectedDecision,
		PolicyID: compiled.Policy.ID, PolicySource: compiled.SourcePath, PolicyDigest: policyDigest,
		AuthorityVector: []CellResult{}, ProofVector: []string{}, IndicatorVector: []string{},
		Unknowns: []Unknown{}, Refutations: []Refutation{}, NextFrontier: []string{},
		FileDigests: map[string]string{}, Authority: manifest.Authority, Release: manifest.Release,
	}
	observations := map[string]CellObservation{}
	duplicateObservation := map[string]bool{}
	for _, observation := range manifest.Cells {
		if _, exists := observations[observation.CellID]; exists {
			duplicateObservation[observation.CellID] = true
		}
		observations[observation.CellID] = observation
	}

	for _, spec := range compiled.Policy.Cells {
		observation, found := observations[spec.ID]
		result := CellResult{
			ID: spec.ID, SemanticRole: spec.SemanticRole, Description: humanDescription(spec.Description),
			Activity: spec.Activity, Proof: spec.Proof, Indicator: spec.Indicator,
			AuthorityState: spec.ExpectedState, Evidence: []EvidenceRecord{},
		}
		report.ProofVector = append(report.ProofVector, spec.Proof)
		report.IndicatorVector = append(report.IndicatorVector, spec.Indicator)
		if duplicateObservation[spec.ID] {
			result.AuthorityState = DecisionRefuted
			appendRefutation(&report, Refutation{
				Stage: "INPUT", Step: "SELECT_AUTHORITY_CELL", Reason: "DUPLICATE_CELL_OBSERVATION",
				Counterexample: "more than one observation claims authority for " + spec.ID, CellID: spec.ID,
			})
			report.AuthorityVector = append(report.AuthorityVector, result)
			continue
		}
		if !found {
			result.AuthorityState = DecisionUnknown
			appendUnknown(&report, Unknown{
				Stage: "BOUNDARY_INPUT", Step: "READ_AUTHORITY_CELL", Reason: "AUTHORITY_CELL_OBSERVATION_MISSING",
				UnknownClass: "DIRECT_MISSING", NextOperation: "PROVIDE_EXPLICIT_CELL_EVIDENCE",
				BlockedBy: []string{"CELL:" + spec.ID},
			})
			report.NextFrontier = append(report.NextFrontier, spec.ID)
			report.AuthorityVector = append(report.AuthorityVector, result)
			continue
		}

		state, records, unknown, refuted := classifyCell(spec, observation, manifestBase, repositoryRoot, &report)
		result.AuthorityState = state
		result.Evidence = records
		if unknown || refuted {
			report.NextFrontier = append(report.NextFrontier, spec.ID)
		}
		report.AuthorityVector = append(report.AuthorityVector, result)
	}

	checkAuthorityCounters(&report)
	checkFixedPoint(&report, manifest.FixedPoint, manifestBase, repositoryRoot)
	report.Decision = reduce(report)
	report.NextFrontier = uniqueSorted(report.NextFrontier)
	return report, nil
}

func classifyCell(spec CellSpec, observation CellObservation, base, root string, report *Report) (string, []EvidenceRecord, bool, bool) {
	if !OwnedAuthorityStates[observation.ClaimedState] {
		appendRefutation(report, Refutation{
			Stage: "BOUNDARY_INPUT", Step: "VALIDATE_AUTHORITY_STATE", Reason: "UNSCOPED_AUTHORITY_STATE",
			Counterexample: "claimed state is not one of the four evidence-backed owner states: " + observation.ClaimedState,
			CellID:         spec.ID,
		})
		return DecisionRefuted, []EvidenceRecord{}, false, true
	}
	if observation.Method == "FILE_EXTENSION_ONLY" || observation.Method == "LINE_COUNT_ONLY" {
		appendRefutation(report, Refutation{
			Stage: "BOUNDARY_INPUT", Step: "CLASSIFY_SEMANTIC_AUTHORITY", Reason: "SEMANTIC_AUTHORITY_INFERRED_FROM_FILE_SHAPE",
			Counterexample: "file extension or line count was used without an explicit semantic marker",
			CellID:         spec.ID,
		})
		return DecisionRefuted, []EvidenceRecord{}, false, true
	}
	if observation.Method != "EXPLICIT_SEMANTIC_MARKER" {
		appendRefutation(report, Refutation{
			Stage: "BOUNDARY_INPUT", Step: "CLASSIFY_SEMANTIC_AUTHORITY", Reason: "NON_EXPLICIT_AUTHORITY_METHOD",
			Counterexample: "authority method must be EXPLICIT_SEMANTIC_MARKER",
			CellID:         spec.ID,
		})
		return DecisionRefuted, []EvidenceRecord{}, false, true
	}
	if len(observation.Evidence) == 0 {
		appendUnknown(report, Unknown{
			Stage: "BOUNDARY_INPUT", Step: "READ_CELL_EVIDENCE", Reason: "CELL_EVIDENCE_MISSING",
			UnknownClass: "DIRECT_MISSING", NextOperation: "PROVIDE_IMMUTABLE_CELL_EVIDENCE",
			BlockedBy: []string{"CELL:" + spec.ID},
		})
		return DecisionUnknown, []EvidenceRecord{}, true, false
	}

	records := make([]EvidenceRecord, 0, len(observation.Evidence))
	seenAuthority := map[string]bool{}
	unknown := false
	refuted := false
	for _, evidence := range observation.Evidence {
		record, status, authority := inspectEvidence(spec, observation, evidence, base, root, report)
		records = append(records, record)
		if record.Digest != nil {
			report.FileDigests[evidence.Path] = *record.Digest
		}
		switch status {
		case DecisionUnknown:
			unknown = true
		case DecisionRefuted:
			refuted = true
		}
		if authority != "" {
			seenAuthority[authority] = true
		}
	}
	if len(seenAuthority) > 1 {
		refuted = true
		appendRefutation(report, Refutation{
			Stage: "BOUNDARY_INPUT", Step: "COMPARE_AUTHORITY_EVIDENCE", Reason: "MULTIPLE_SEMANTIC_AUTHORITIES",
			Counterexample: "evidence records declare different authority states for " + spec.ID,
			CellID:         spec.ID,
		})
	}
	if unknown {
		return DecisionUnknown, records, true, refuted
	}
	if refuted {
		return DecisionRefuted, records, false, true
	}
	if len(seenAuthority) == 0 {
		return DecisionUnknown, records, true, false
	}
	for authority := range seenAuthority {
		if authority != observation.ClaimedState {
			appendRefutation(report, Refutation{
				Stage: "BOUNDARY_INPUT", Step: "COMPARE_DECLARED_AUTHORITY", Reason: "AUTHORITY_MARKER_CONTRADICTION",
				Counterexample: "claimed=" + observation.ClaimedState + " evidence=" + authority,
				CellID:         spec.ID,
			})
			return DecisionRefuted, records, false, true
		}
	}
	return observation.ClaimedState, records, false, false
}

func inspectEvidence(spec CellSpec, observation CellObservation, evidence EvidenceRef, base, root string, report *Report) (EvidenceRecord, string, string) {
	record := EvidenceRecord{
		Role: evidence.Role, Kind: evidence.Kind, Path: evidence.Path,
		StartLine: evidence.StartLine, EndLine: evidence.EndLine, Digest: nil,
		Lines: []string{}, Status: "MISSING",
	}
	path, err := resolveEvidencePath(base, root, evidence.Path)
	if err != nil {
		record.Status = "REFUTED"
		appendRefutation(report, Refutation{
			Stage: "EVIDENCE", Step: "RESOLVE_IMMUTABLE_INPUT", Reason: "EVIDENCE_PATH_OUTSIDE_INPUT_ROOT",
			Counterexample: evidence.Path, CellID: spec.ID,
		})
		return record, DecisionRefuted, ""
	}
	digest, err := fileDigest(path)
	if err != nil {
		return record, appendEvidenceUnknown(report, spec.ID, "EVIDENCE", "READ_EVIDENCE_FILE", "EVIDENCE_FILE_MISSING", "PROVIDE_RELEASED_INPUT_FILE", []string{"FILE:" + evidence.Path}), ""
	}
	record.Digest = &digest
	data, err := os.ReadFile(path)
	if err != nil {
		return record, appendEvidenceUnknown(report, spec.ID, "EVIDENCE", "READ_EVIDENCE_FILE", "EVIDENCE_FILE_UNREADABLE", "PROVIDE_READABLE_INPUT_FILE", []string{"FILE:" + evidence.Path}), ""
	}
	if evidence.ExpectedDigest != "" && evidence.ExpectedDigest != digest {
		record.Status = "REFUTED"
		appendRefutation(report, Refutation{
			Stage: "EVIDENCE", Step: "VERIFY_INPUT_DIGEST", Reason: "IMMUTABLE_DIGEST_MISMATCH",
			Counterexample: "expected=" + evidence.ExpectedDigest + " actual=" + digest, CellID: spec.ID,
		})
		return record, DecisionRefuted, authorityFromText(string(data))
	}
	lines := splitLines(data)
	if evidence.StartLine < 1 || evidence.EndLine < evidence.StartLine || evidence.EndLine > len(lines) {
		return record, appendEvidenceUnknown(report, spec.ID, "EVIDENCE", "SELECT_EVIDENCE_LINES", "EVIDENCE_LINE_RANGE_UNAVAILABLE", "PIN_RELEASED_LINE_RANGE", []string{"FILE:" + evidence.Path}), ""
	}
	record.Lines = append([]string(nil), lines[evidence.StartLine-1:evidence.EndLine]...)
	selected := strings.Join(record.Lines, "\n")
	if evidence.Marker != "" && !strings.Contains(selected, evidence.Marker) {
		return record, appendEvidenceUnknown(report, spec.ID, "EVIDENCE", "MATCH_EXPLICIT_MARKER", "EVIDENCE_MARKER_UNAVAILABLE", "PIN_EXACT_SEMANTIC_MARKER", []string{"CELL:" + spec.ID}), ""
	}
	if !hasCellKey(selected, spec.ID) || !hasKeyValue(selected, "semantic_role", spec.SemanticRole) || !hasKeyValue(selected, "evidence_kind", spec.EvidenceKind) {
		return record, appendEvidenceUnknown(report, spec.ID, "EVIDENCE", "MATCH_SEMANTIC_IDENTITY", "SEMANTIC_IDENTITY_MARKER_UNAVAILABLE", "PROVIDE_CELL_ROLE_AND_EVIDENCE_KIND_MARKERS", []string{"CELL:" + spec.ID}), ""
	}
	authority := authorityFromText(selected)
	if authority == "" {
		return record, appendEvidenceUnknown(report, spec.ID, "EVIDENCE", "READ_AUTHORITY_MARKER", "AUTHORITY_MARKER_UNAVAILABLE", "PROVIDE_EXPLICIT_AUTHORITY_MARKER", []string{"CELL:" + spec.ID}), ""
	}
	if authority != observation.ClaimedState {
		record.Status = "REFUTED"
		appendRefutation(report, Refutation{
			Stage: "EVIDENCE", Step: "COMPARE_AUTHORITY_MARKER", Reason: "AUTHORITY_MARKER_CONTRADICTION",
			Counterexample: "claimed=" + observation.ClaimedState + " evidence=" + authority, CellID: spec.ID,
		})
		return record, DecisionRefuted, authority
	}
	record.Status = "VERIFIED"
	return record, "", authority
}

func checkAuthorityCounters(report *Report) {
	if report.Authority.RepositoryWrites == 0 && report.Authority.LocalTestExecutions == 0 &&
		report.Authority.InfrastructureMutations == 0 && report.Authority.ProviderInstallAttempts == 0 &&
		report.Authority.NetworkMutationAttempts == 0 {
		return
	}
	appendRefutation(report, Refutation{
		Stage: "AUTHORITY", Step: "CHECK_READ_ONLY_BOUNDARY", Reason: "INPUT_REPOSITORY_WRITE_AUTHORITY_NONZERO",
		Counterexample: "the input authority counters contain a non-zero mutation or local-execution value",
	})
}

func checkFixedPoint(report *Report, evidence *EvidenceRef, base, root string) {
	if evidence == nil {
		report.FixedPoint = FixedPointResult{State: DecisionUnknown}
		appendUnknown(report, Unknown{
			Stage: "FIXED_POINT", Step: "READ_EXPLICIT_FIXED_POINT", Reason: "FIXED_POINT_EVIDENCE_ABSENT",
			UnknownClass: "EXPLICIT_ONLY", NextOperation: "PROVIDE_EXPLICIT_FIXED_POINT_MARKER",
			BlockedBy: []string{"FIXED_POINT_EVIDENCE"},
		})
		return
	}
	record := EvidenceRecord{Role: evidence.Role, Kind: evidence.Kind, Path: evidence.Path, StartLine: evidence.StartLine, EndLine: evidence.EndLine, Digest: nil, Lines: []string{}, Status: "MISSING"}
	path, err := resolveEvidencePath(base, root, evidence.Path)
	if err != nil {
		report.FixedPoint = FixedPointResult{State: DecisionRefuted, Evidence: &record}
		appendRefutation(report, Refutation{Stage: "FIXED_POINT", Step: "RESOLVE_EVIDENCE", Reason: "FIXED_POINT_PATH_OUTSIDE_INPUT_ROOT", Counterexample: evidence.Path})
		return
	}
	digest, err := fileDigest(path)
	if err != nil {
		report.FixedPoint = FixedPointResult{State: DecisionUnknown, Evidence: &record}
		appendUnknown(report, Unknown{Stage: "FIXED_POINT", Step: "READ_EXPLICIT_FIXED_POINT", Reason: "FIXED_POINT_FILE_MISSING", UnknownClass: "DIRECT_MISSING", NextOperation: "PROVIDE_EXPLICIT_FIXED_POINT_MARKER", BlockedBy: []string{"FILE:" + evidence.Path}})
		return
	}
	record.Digest = &digest
	data, _ := os.ReadFile(path)
	lines := splitLines(data)
	if evidence.StartLine < 1 || evidence.EndLine < evidence.StartLine || evidence.EndLine > len(lines) {
		report.FixedPoint = FixedPointResult{State: DecisionUnknown, Evidence: &record}
		appendUnknown(report, Unknown{Stage: "FIXED_POINT", Step: "SELECT_EXPLICIT_MARKER", Reason: "FIXED_POINT_LINE_RANGE_UNAVAILABLE", UnknownClass: "SCHEMA_MISSING", NextOperation: "PIN_EXPLICIT_FIXED_POINT_LINE", BlockedBy: []string{"FILE:" + evidence.Path}})
		return
	}
	record.Lines = append([]string(nil), lines[evidence.StartLine-1:evidence.EndLine]...)
	selected := strings.Join(record.Lines, "\n")
	if evidence.Marker == "" || !strings.Contains(selected, evidence.Marker) || !strings.Contains(selected, "state=FIXED_POINT") {
		record.Status = "REFUTED"
		report.FixedPoint = FixedPointResult{State: DecisionRefuted, Evidence: &record}
		appendRefutation(report, Refutation{Stage: "FIXED_POINT", Step: "COMPARE_EXPLICIT_MARKER", Reason: "FIXED_POINT_MARKER_CONTRADICTION", Counterexample: "an explicit FIXED_POINT claim did not match the selected immutable line"})
		return
	}
	record.Status = "VERIFIED"
	report.FixedPoint = FixedPointResult{State: FixedPoint, Evidence: &record}
	report.FileDigests[evidence.Path] = digest
}

func appendEvidenceUnknown(report *Report, cellID, stage, step, reason, next string, blocked []string) string {
	appendUnknown(report, Unknown{Stage: stage, Step: step, Reason: reason, UnknownClass: "DIRECT_MISSING", NextOperation: next, BlockedBy: blocked})
	if cellID == "" {
		return DecisionUnknown
	}
	return DecisionUnknown
}

func appendUnknown(report *Report, unknown Unknown) {
	report.Unknowns = append(report.Unknowns, unknown)
}

func appendRefutation(report *Report, refutation Refutation) {
	report.Refutations = append(report.Refutations, refutation)
}

func reduce(report Report) string {
	if len(report.Refutations) > 0 {
		return DecisionRefuted
	}
	if len(report.Unknowns) > 0 {
		return DecisionUnknown
	}
	return DecisionClosed
}

func replayResult(first, second Report) ReplayResult {
	firstDigest := projectionDigest(first)
	secondDigest := projectionDigest(second)
	mismatches := 0
	if firstDigest != secondDigest {
		mismatches = 1
	}
	state := DecisionClosed
	if mismatches > 0 {
		state = DecisionRefuted
	}
	return ReplayResult{Comparisons: 1, Mismatches: mismatches, ProjectionDigest: firstDigest, ReplayDigest: secondDigest, State: state}
}

func projectionDigest(report Report) string {
	projection := struct {
		Decision        string            `json:"decision"`
		AuthorityVector []CellResult      `json:"authority_vector"`
		ProofVector     []string          `json:"proof_vector"`
		IndicatorVector []string          `json:"indicator_vector"`
		FixedPoint      FixedPointResult  `json:"fixed_point"`
		Unknowns        []Unknown         `json:"unknowns"`
		Refutations     []Refutation      `json:"refutations"`
		NextFrontier    []string          `json:"next_frontier"`
		FileDigests     map[string]string `json:"file_digests"`
	}{report.Decision, report.AuthorityVector, report.ProofVector, report.IndicatorVector, report.FixedPoint, report.Unknowns, report.Refutations, report.NextFrontier, report.FileDigests}
	data, _ := json.Marshal(projection)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeReport(outputDir string, report Report) error {
	if !filepath.IsAbs(outputDir) {
		return errors.New("output directory must be an absolute caller-owned path")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	jsonBytes = append(jsonBytes, '\n')
	if err := os.WriteFile(filepath.Join(outputDir, "authority-boundary-report.json"), jsonBytes, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "authority-boundary-report.md"), []byte(RenderHumanReport(report)), 0o644)
}

func RenderHumanReport(report Report) string {
	var b strings.Builder
	b.WriteString("# Gooo self-description boundary\n\n")
	fmt.Fprintf(&b, "- Decision: `%s`\n", report.Decision)
	fmt.Fprintf(&b, "- Fixed authority-cell denominator: %d\n", len(report.AuthorityVector))
	fmt.Fprintf(&b, "- Explicit fixed-point observation: `%s`\n", report.FixedPoint.State)
	b.WriteString("- Aggregate score and percentage: not emitted\n")
	b.WriteString("\n| Cell | Semantic role | Authority state | Proof | Indicator | Evidence |\n|---|---|---|---|---|---|\n")
	for _, cell := range report.AuthorityVector {
		evidence := "none"
		if len(cell.Evidence) > 0 {
			parts := make([]string, 0, len(cell.Evidence))
			for _, item := range cell.Evidence {
				parts = append(parts, fmt.Sprintf("%s:%d-%d (%s)", item.Path, item.StartLine, item.EndLine, item.Status))
			}
			evidence = strings.Join(parts, "; ")
		}
		fmt.Fprintf(&b, "| `%s` | %s | `%s` | `%s` | `%s` | %s |\n", cell.ID, cell.Description, cell.AuthorityState, cell.Proof, cell.Indicator, evidence)
	}
	b.WriteString("\nProof and indicator are independent declared dimensions. Authority is accepted only from explicit semantic markers with immutable file digests; file extension and line count alone are refuted. Go 1.27 is the read-only projector/evaluator/generator runtime.\n")
	return b.String()
}

func resolveEvidencePath(base, root, relative string) (string, error) {
	if relative == "" {
		return "", errors.New("empty evidence path")
	}
	path, err := filepath.Abs(filepath.Join(base, filepath.Clean(relative)))
	if err != nil {
		return "", err
	}
	relativeToRoot, err := filepath.Rel(root, path)
	if err != nil || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(os.PathSeparator)) {
		return "", errors.New("evidence path outside repository root")
	}
	return path, nil
}

func splitLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

func authorityFromText(text string) string {
	found := []string{}
	for _, state := range AuthorityStates {
		if strings.Contains(text, "authority="+state) || strings.Contains(text, `"authority":"`+state+`"`) || strings.Contains(text, "authority "+state) {
			found = append(found, state)
		}
	}
	if len(found) != 1 {
		return ""
	}
	return found[0]
}

func hasCellKey(text, id string) bool {
	return strings.Contains(text, "cell="+id) || strings.Contains(text, `"cell":"`+id+`"`) || strings.Contains(text, "cell "+id)
}

func hasKeyValue(text, key, value string) bool {
	return strings.Contains(text, key+"="+value) || strings.Contains(text, `"`+key+`":"`+value+`"`) || strings.Contains(text, key+" "+value)
}

func humanDescription(description string) string {
	return strings.ReplaceAll(description, "_", " ")
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	sort.Strings(values)
	out := []string{values[0]}
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
