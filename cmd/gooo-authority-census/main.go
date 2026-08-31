package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-semantic-authority-census/internal/census"
)

func main() {
	policy := flag.String("policy", "", "compiled policy JSON")
	manifest := flag.String("manifest", "", "scenario manifest JSON")
	out := flag.String("out", "", "caller-owned report path")
	flag.Parse()
	if *policy == "" || *manifest == "" || *out == "" {
		fail("--policy, --manifest, and --out are required")
	}
	report, err := census.Evaluate(*policy, *manifest)
	if err != nil {
		fail(err.Error())
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail(err.Error())
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err.Error())
	}
	fmt.Printf("%s\t%s\n", report.ScenarioID, report.Decision)
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
