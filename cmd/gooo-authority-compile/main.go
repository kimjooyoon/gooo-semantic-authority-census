package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type policy struct {
	Schema        string   `json:"schema"`
	ID            string   `json:"id"`
	Precedence    []string `json:"precedence"`
	UnknownFields []string `json:"unknown_fields"`
	Cells         []cell   `json:"cells"`
}

type cell struct {
	ID        string `json:"id"`
	Proof     string `json:"proof"`
	Indicator string `json:"indicator"`
	Activity  string `json:"activity"`
}

func main() {
	source := flag.String("source", "", "Gooo policy source")
	out := flag.String("out", "", "caller-owned output directory")
	flag.Parse()
	if *source == "" || *out == "" {
		fail(errors.New("--source and --out are required"))
	}
	p, err := parse(*source)
	if err != nil {
		fail(err)
	}
	if err := validate(p); err != nil {
		fail(err)
	}
	encoded, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		fail(err)
	}
	encoded = append(encoded, '\n')
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	irPath := filepath.Join(*out, "policy.ir.json")
	if err := os.WriteFile(irPath, encoded, 0o644); err != nil {
		fail(err)
	}
	generated := fmt.Sprintf("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Print(%s)\n}\n", strconv.Quote(string(encoded)))
	goPath := filepath.Join(*out, "policy_generated.go")
	if err := os.WriteFile(goPath, []byte(generated), 0o644); err != nil {
		fail(err)
	}
	summary := map[string]any{
		"policy_id": p.ID,
		"cells": len(p.Cells),
		"ir_path": irPath,
		"generated_path": goPath,
		"repository_writes": 0,
	}
	outBytes, _ := json.Marshal(summary)
	fmt.Println(string(outBytes))
}

func parse(path string) (policy, error) {
	f, err := os.Open(path)
	if err != nil {
		return policy{}, err
	}
	defer f.Close()
	p := policy{Schema: "gooo/semantic-authority-policy/v1"}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "policy":
			if len(fields) != 2 {
				return policy{}, fmt.Errorf("invalid policy line: %s", line)
			}
			p.ID = fields[1]
		case "precedence":
			if len(fields) != 4 {
				return policy{}, fmt.Errorf("invalid precedence line: %s", line)
			}
			p.Precedence = append([]string(nil), fields[1:]...)
		case "unknown_fields":
			if len(fields) != 7 {
				return policy{}, fmt.Errorf("invalid unknown_fields line: %s", line)
			}
			p.UnknownFields = append([]string(nil), fields[1:]...)
		case "cell":
			if len(fields) != 5 {
				return policy{}, fmt.Errorf("invalid cell line: %s", line)
			}
			p.Cells = append(p.Cells, cell{ID: fields[1], Proof: fields[2], Indicator: fields[3], Activity: fields[4]})
		default:
			return policy{}, fmt.Errorf("unknown Gooo directive: %s", fields[0])
		}
	}
	return p, scanner.Err()
}

func validate(p policy) error {
	if p.ID == "" {
		return errors.New("policy id is missing")
	}
	if strings.Join(p.Precedence, ",") != "REFUTED,UNKNOWN,CLOSED" {
		return errors.New("precedence must be REFUTED UNKNOWN CLOSED")
	}
	if strings.Join(p.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return errors.New("UNKNOWN fields are not the fixed six-field schema")
	}
	if len(p.Cells) != 12 {
		return fmt.Errorf("expected 12 cells, got %d", len(p.Cells))
	}
	proofs := map[string]int{}
	indicators := map[string]int{}
	ids := map[string]bool{}
	activities := map[string]bool{}
	for _, c := range p.Cells {
		if ids[c.ID] || activities[c.Activity] {
			return fmt.Errorf("duplicate cell or activity: %s", c.ID)
		}
		ids[c.ID] = true
		activities[c.Activity] = true
		proofs[c.Proof]++
		indicators[c.Indicator]++
	}
	for _, k := range []string{"FOUNDATION", "COHERENCE", "REGRESSION"} {
		if proofs[k] != 4 {
			return fmt.Errorf("proof %s count is %d", k, proofs[k])
		}
	}
	for _, k := range []string{"DRIVER", "OUTCOME", "GUARDRAIL"} {
		if indicators[k] != 4 {
			return fmt.Errorf("indicator %s count is %d", k, indicators[k])
		}
	}
	return nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
