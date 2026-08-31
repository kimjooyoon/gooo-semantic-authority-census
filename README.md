# Gooo Semantic Authority Census

This repository measures an exact semantic-authority frontier. It does not assign a score and does not claim that Gooo is self-hosting.

A Gooo policy defines twelve census cells, proof and indicator partitions, decision precedence, and the six required UNKNOWN fields. CI compiles that policy to semantic IR and generated Go in a caller-owned temporary directory. The census then classifies each semantic obligation as generated from Gooo, handwritten Go, UNKNOWN, or REFUTED.

The canonical corpus contains exactly nine cases:

- 3 CLOSED observations
- 3 UNKNOWN observations
- 3 REFUTED observations

The paired fixture records generated bindings 2 -> 3 and handwritten bindings 1 -> 0 under the same three-obligation denominator. These are exact counts, not a percentage or a general improvement claim.

All build, test, conformance, replay, inventory, time, and RSS observations run in GitHub Actions with Go 1.27. Root README.md is excluded from inventory counts. Repository-write authority, local test execution, infrastructure mutation, provider installation, and network mutation are all zero.

The handwritten Go compiler and evaluator remain an explicit bootstrap boundary. This release can expose the next migration frontier; it cannot by itself close CORE_SEMANTIC_AUTHORITY for another repository.
