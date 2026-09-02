# Gooo semantic authority census

This repository contains two deliberately separate evidence axes:

1. The original operational semantic census in `meta/authority-census.gooo`.
2. The independent Gooo self-description boundary projector in
   `meta/self-description-boundary.gooo`.

The second axis is the requested boundary record. It does not calculate a
self-hosting percentage, improvement score, or aggregate ownership claim. It
projects a fixed denominator of exactly eight semantic authority cells:

`LANGUAGE_GRAMMAR`, `BOUNDARY_ONTOLOGY`, `SEMANTIC_IR_SCHEMA`,
`EVALUATOR_RULES`, `GENERATOR_RULES`, `CONFORMANCE_POLICY`,
`TOOLCHAIN_METADATA`, and `GENERATED_GO_LINEAGE`.

Each cell is classified from exact evidence as one of:

`GOOO_OWNED`, `GENERATED_FROM_GOOO`, `HANDWRITTEN_RUNTIME`,
`BOOTSTRAP_EXTERNAL`, `UNKNOWN`, or `REFUTED`.

The released `.gooo` policy owns the denominator, authority vocabulary,
semantic roles, meta activities, proof dimension, indicator dimension,
`REFUTED > UNKNOWN > CLOSED` precedence, six-field UNKNOWN schema,
`EXPLICIT_ONLY` fixed-point rule, and `READ_ONLY` output boundary. Go 1.27 is
only the parser/compiler, read-only projector/evaluator, and generated-policy
runtime. It does not become semantic authority merely because it has more
files or lines.

The projector accepts released Gooo source, semantic graph, generated Go, and
toolchain metadata. Evidence must name the cell, semantic role, evidence kind,
and authority explicitly. File extensions and line counts alone are
`REFUTED`. Every selected evidence record carries an exact path, line range,
selected text, and SHA-256 digest. `FIXED_POINT` is accepted only when an
explicit `state=FIXED_POINT` marker is present.

The boundary corpus is exact: three normal (`CLOSED`), three `UNKNOWN`, and
three `REFUTED` scenarios. The reports contain authority/proof/indicator
vectors and human-readable per-cell explanations, never a score or
percentage. Missing observations retain `null` measurements and six-field
UNKNOWN evidence.

GitHub Actions is the validation boundary. It uses Go 1.27, compiles the
`.gooo` policy to semantic IR and generated Go, builds the projector, runs the
tests, projects all nine cases, captures per-stage build/test/compile/conformance
wall time and peak RSS, cache status, exact Go/`.gooo` inventory, replay
digests, and run identity. The root `README.md` is excluded from regular-file
inventory counts. The projector writes only to an absolute caller-owned
temporary output directory; repository-write authority is zero.

Failed Actions runs upload their partial caller-owned output with a unique
run-scoped artifact name. After a green pull-request and main-branch run, the
release workflow creates a draft patch release, uploads source/evidence assets,
records SHA-256 digests, publishes once, and verifies the immutable release.
Tags, releases, failed runs, and evidence are never deleted or reused.

The detailed protocol is in
[`docs/self-description-boundary-v1.md`](docs/self-description-boundary-v1.md).
