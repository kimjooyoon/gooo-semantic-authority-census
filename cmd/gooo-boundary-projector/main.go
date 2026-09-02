package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-semantic-authority-census/internal/boundary"
)

func main() {
	policy := flag.String("policy", "", "compiled .gooo boundary policy JSON")
	manifest := flag.String("manifest", "", "released input/evidence manifest JSON")
	output := flag.String("output", "", "absolute caller-owned output directory")
	root := flag.String("root", ".", "repository input root")
	flag.Parse()
	if *policy == "" || *manifest == "" || *output == "" {
		fail("--policy, --manifest, and --output are required")
	}
	report, err := boundary.Project(*policy, *manifest, *output, *root)
	if err != nil {
		fail(err.Error())
	}
	data, err := json.Marshal(map[string]any{
		"schema": boundary.ReportSchema + "/receipt",
		"scenario_id": report.ScenarioID,
		"decision": report.Decision,
		"authority_cells": len(report.AuthorityVector),
		"repository_writes": report.Authority.RepositoryWrites,
		"replay": report.Replay.State,
	})
	if err != nil {
		fail(err.Error())
	}
	fmt.Println(string(data))
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
