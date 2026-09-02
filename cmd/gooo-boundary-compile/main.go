package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-semantic-authority-census/internal/boundary"
)

func main() {
	source := flag.String("source", "", "released .gooo boundary policy")
	out := flag.String("out", "", "caller-owned output directory")
	flag.Parse()
	if *source == "" || *out == "" {
		fail("--source and --out are required")
	}
	compiled, err := boundary.CompilePolicy(*source, *out)
	if err != nil {
		fail(err.Error())
	}
	data, err := json.Marshal(map[string]any{
		"schema":            boundary.PolicySchema + "/compile-receipt",
		"policy_id":         compiled.Policy.ID,
		"authority_cells":   len(compiled.Policy.Cells),
		"source_digest":     compiled.SourceDigest,
		"repository_writes": 0,
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
