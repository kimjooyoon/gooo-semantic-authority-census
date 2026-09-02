# Gooo self-description boundary protocol v1

This protocol is an independent axis from any self-hosting, improvement, or
utility claim. It answers a narrower question: for each fixed compiler or
toolchain semantic obligation, which authority is evidenced by the released
inputs?

## Fixed denominator

The denominator is exactly eight authority cells:

1. `LANGUAGE_GRAMMAR`
2. `BOUNDARY_ONTOLOGY`
3. `SEMANTIC_IR_SCHEMA`
4. `EVALUATOR_RULES`
5. `GENERATOR_RULES`
6. `CONFORMANCE_POLICY`
7. `TOOLCHAIN_METADATA`
8. `GENERATED_GO_LINEAGE`

The `.gooo` policy declares the stable cell IDs, expected authority vocabulary,
meta activities, proof dimension, indicator dimension, fixed-point rule, and
the output boundary. Proof and indicator are carried independently in every
cell; neither is inferred from the other or from authority state.

## Authority states

- `GOOO_OWNED`: explicit released `.gooo` semantic content names the rule.
- `GENERATED_FROM_GOOO`: generated Go contains an explicit lineage marker.
- `HANDWRITTEN_RUNTIME`: handwritten Go explicitly names the runtime rule.
- `BOOTSTRAP_EXTERNAL`: external toolchain metadata explicitly names the rule.
- `UNKNOWN`: the required immutable evidence is absent, ambiguous, or not
  selectable. The record always contains exactly the six UNKNOWN fields.
- `REFUTED`: evidence contradicts the claim, authority is escalated, or the
  method relies on file extension/line count instead of semantic content.

Decision precedence is `REFUTED > UNKNOWN > CLOSED`. `CLOSED` means the case
was evaluated without a refutation or unresolved evidence; it is not a score.
No percentage or aggregate self-hosted/improved claim is emitted.

## Inputs and evidence

The projector accepts a compiled policy plus an input manifest describing a
released Gooo source file, semantic graph, generated Go, and toolchain
metadata. Every selected evidence record contains path, exact line range,
selected line text, and a SHA-256 digest computed from the file in the same
Actions run. A line is accepted only when it contains the cell identity,
semantic role, evidence kind, and explicit authority marker. File suffixes,
file counts, and line counts have no semantic force; an extension-only case is
explicitly `REFUTED`.

`FIXED_POINT` is accepted only from a selected line containing the literal
`state=FIXED_POINT`. No fixed point is inferred from a closed case or from a
matching generated file.

## Runtime boundary

Go 1.27 is the read-only projector, evaluator, and generator runtime. It reads
the released inputs and writes only to an absolute caller-owned output
directory outside the repository. The input manifest must report zero
repository writes, local test executions, infrastructure mutations, provider
install attempts, and network mutation attempts; a non-zero counter is
`REFUTED`.

## Corpus and CI evidence

The boundary corpus contains exactly three normal (`CLOSED`), three
`UNKNOWN`, and three `REFUTED` scenarios. GitHub Actions compiles the `.gooo`
policy, compares its generated Go policy output with the semantic IR, builds
the projector, runs tests, and projects the corpus. It records per-stage wall
time and peak RSS, cache status, exact Go/`.gooo` file paths and physical-line
counts, exact descendant directories, run identity, and all file digests.

When an observation is not available, its value is `null` and its measurement
state is `UNKNOWN`; no zero is substituted. Failed runs upload the partial
caller-owned output with `if: always()` and run-specific artifact names.
