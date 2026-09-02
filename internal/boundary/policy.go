package boundary

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ParsePolicy(path string) (Policy, error) {
	file, err := os.Open(path)
	if err != nil {
		return Policy{}, err
	}
	defer file.Close()

	policy := Policy{Schema: PolicySchema}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "boundary_policy":
			if len(fields) != 2 {
				return Policy{}, fmt.Errorf("invalid boundary_policy line: %s", line)
			}
			policy.ID = fields[1]
		case "release":
			if len(fields) != 2 {
				return Policy{}, fmt.Errorf("invalid release line: %s", line)
			}
			policy.Release = fields[1]
		case "precedence":
			if len(fields) != 4 {
				return Policy{}, fmt.Errorf("invalid precedence line: %s", line)
			}
			policy.Precedence = append([]string(nil), fields[1:]...)
		case "unknown_fields":
			if len(fields) != 7 {
				return Policy{}, fmt.Errorf("invalid unknown_fields line: %s", line)
			}
			policy.UnknownFields = append([]string(nil), fields[1:]...)
		case "authority_states":
			if len(fields) != 7 {
				return Policy{}, fmt.Errorf("invalid authority_states line: %s", line)
			}
			policy.AuthorityStates = append([]string(nil), fields[1:]...)
		case "fixed_point_rule":
			if len(fields) != 2 {
				return Policy{}, fmt.Errorf("invalid fixed_point_rule line: %s", line)
			}
			policy.FixedPointRule = fields[1]
		case "output_authority":
			if len(fields) != 2 {
				return Policy{}, fmt.Errorf("invalid output_authority line: %s", line)
			}
			policy.OutputAuthority = fields[1]
		case "authority_cell":
			if len(fields) < 9 {
				return Policy{}, fmt.Errorf("invalid authority_cell line: %s", line)
			}
			policy.Cells = append(policy.Cells, CellSpec{
				ID: fields[1], ExpectedState: fields[2], EvidenceKind: fields[3],
				SemanticRole: fields[4], Activity: fields[5], Proof: fields[6],
				Indicator: fields[7], Description: strings.Join(fields[8:], " "),
			})
		default:
			return Policy{}, fmt.Errorf("unknown boundary directive: %s", fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Policy{}, err
	}
	return policy, ValidatePolicy(policy)
}

func ValidatePolicy(policy Policy) error {
	if policy.Schema != "" && policy.Schema != PolicySchema {
		return fmt.Errorf("unexpected policy schema %q", policy.Schema)
	}
	if policy.ID == "" || policy.Release == "" {
		return errors.New("policy id and release are required")
	}
	if strings.Join(policy.Precedence, ",") != "REFUTED,UNKNOWN,CLOSED" {
		return errors.New("precedence must be REFUTED UNKNOWN CLOSED")
	}
	if strings.Join(policy.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return errors.New("UNKNOWN fields must be the fixed six-field schema")
	}
	if strings.Join(policy.AuthorityStates, ",") != strings.Join(AuthorityStates, ",") {
		return errors.New("authority states are not the fixed six-state vocabulary")
	}
	if policy.FixedPointRule != "EXPLICIT_ONLY" {
		return errors.New("fixed point rule must be EXPLICIT_ONLY")
	}
	if policy.OutputAuthority != "READ_ONLY" {
		return errors.New("output authority must be READ_ONLY")
	}
	if len(policy.Cells) != 8 {
		return fmt.Errorf("authority-cell denominator is %d, expected 8", len(policy.Cells))
	}
	ids := map[string]bool{}
	activities := map[string]bool{}
	for _, cell := range policy.Cells {
		if cell.ID == "" || ids[cell.ID] {
			return fmt.Errorf("duplicate or empty authority cell %q", cell.ID)
		}
		if cell.Activity == "" || activities[cell.Activity] {
			return fmt.Errorf("duplicate or empty meta activity %q", cell.Activity)
		}
		if !OwnedAuthorityStates[cell.ExpectedState] {
			return fmt.Errorf("authority cell %s has invalid expected state %q", cell.ID, cell.ExpectedState)
		}
		if cell.EvidenceKind == "" || cell.SemanticRole == "" || cell.Proof == "" || cell.Indicator == "" {
			return fmt.Errorf("authority cell %s is missing an independent proof/indicator/evidence dimension", cell.ID)
		}
		ids[cell.ID] = true
		activities[cell.Activity] = true
	}
	return nil
}

func CompilePolicy(sourcePath, outputDir string) (CompiledPolicy, error) {
	policy, err := ParsePolicy(sourcePath)
	if err != nil {
		return CompiledPolicy{}, err
	}
	digest, err := fileDigest(sourcePath)
	if err != nil {
		return CompiledPolicy{}, err
	}
	compiled := CompiledPolicy{Schema: PolicySchema, SourcePath: sourcePath, SourceDigest: digest, Policy: policy}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return CompiledPolicy{}, err
	}
	encoded, err := json.MarshalIndent(compiled, "", "  ")
	if err != nil {
		return CompiledPolicy{}, err
	}
	encoded = append(encoded, '\n')
	irPath := filepath.Join(outputDir, "boundary-policy.ir.json")
	if err := os.WriteFile(irPath, encoded, 0o644); err != nil {
		return CompiledPolicy{}, err
	}
	generated := fmt.Sprintf("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Print(%s)\n}\n", strconv.Quote(string(encoded)))
	if err := os.WriteFile(filepath.Join(outputDir, "boundary-policy.generated.go"), []byte(generated), 0o644); err != nil {
		return CompiledPolicy{}, err
	}
	return compiled, nil
}

func LoadCompiledPolicy(path string) (CompiledPolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CompiledPolicy{}, err
	}
	var compiled CompiledPolicy
	if err := json.Unmarshal(data, &compiled); err != nil {
		return CompiledPolicy{}, err
	}
	if compiled.Schema != PolicySchema {
		return CompiledPolicy{}, fmt.Errorf("unexpected compiled policy schema %q", compiled.Schema)
	}
	if err := ValidatePolicy(compiled.Policy); err != nil {
		return CompiledPolicy{}, err
	}
	return compiled, nil
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
