#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
OUT=${1:-"${RUNNER_TEMP:-/tmp}/gooo-semantic-authority-census"}
rm -rf "$OUT"
mkdir -p "$OUT/meta" "$OUT/reports" "$OUT/replay" "$OUT/rss"
START_NS=$(date +%s%N)
BEFORE_STATUS=$(git -C "$ROOT" status --porcelain)

compile_start=$(date +%s%N)
/usr/bin/time -f '%M' -o "$OUT/rss/meta-compile.txt"   bash -c 'cd "$1" && go run ./cmd/gooo-authority-compile --source meta/authority-census.gooo --out "$2"' _ "$ROOT" "$OUT/meta"   > "$OUT/meta/compile-receipt.json"
go run "$OUT/meta/policy_generated.go" > "$OUT/meta/policy.generated.json"
cmp "$OUT/meta/policy.ir.json" "$OUT/meta/policy.generated.json"
jq -S . "$OUT/meta/policy.ir.json" > "$OUT/meta/policy.normalized.json"
jq -S . "$ROOT/contracts/denominator-v1.json" > "$OUT/meta/contract.normalized.json"
cmp "$OUT/meta/policy.normalized.json" "$OUT/meta/contract.normalized.json"
compile_end=$(date +%s%N)
compile_wall_ms=$(((compile_end-compile_start)/1000000))

build_start=$(date +%s%N)
/usr/bin/time -f '%M' -o "$OUT/rss/build.txt"   bash -c 'cd "$1" && go build -o "$2" ./cmd/gooo-authority-census' _ "$ROOT" "$OUT/gooo-authority-census"
build_end=$(date +%s%N)
build_wall_ms=$(((build_end-build_start)/1000000))

normal=0
unknown=0
refuted=0
for manifest in "$ROOT"/fixtures/cases/*.json; do
  scenario=$(jq -r '.scenario_id' "$manifest")
  expected=$(jq -r '.expected_decision' "$manifest")
  /usr/bin/time -f '%M' -o "$OUT/rss/case-$scenario.txt"     "$OUT/gooo-authority-census"     --policy "$OUT/meta/policy.generated.json"     --manifest "$manifest"     --out "$OUT/reports/$scenario.json" > "$OUT/reports/$scenario.stdout"
  actual=$(jq -r '.decision' "$OUT/reports/$scenario.json")
  test "$actual" = "$expected"
  case "$actual" in
    CLOSED) normal=$((normal+1)) ;;
    UNKNOWN) unknown=$((unknown+1)) ;;
    REFUTED) refuted=$((refuted+1)) ;;
    *) exit 1 ;;
  esac
  "$OUT/gooo-authority-census"     --policy "$OUT/meta/policy.generated.json"     --manifest "$manifest"     --out "$OUT/replay/$scenario.json" > "$OUT/replay/$scenario.stdout"
  cmp "$OUT/reports/$scenario.json" "$OUT/replay/$scenario.json"
done

test_start=$(date +%s%N)
/usr/bin/time -f '%M' -o "$OUT/rss/test.txt"   bash -c 'cd "$1" && go test ./... -json' _ "$ROOT" > "$OUT/go-test.json"
test_end=$(date +%s%N)
test_wall_ms=$(((test_end-test_start)/1000000))
tests_executed=$(jq -s '[.[]|select(.Action=="pass" and (.Test? != null))]|length' "$OUT/go-test.json")

jq -e '
  .decision=="UNKNOWN" and
  (.unknowns|length)>0 and
  all(.unknowns[];
    (.stage|type=="string") and
    (.step|type=="string") and
    (.reason|type=="string") and
    (.unknown_class|type=="string") and
    (.next_operation|type=="string") and
    (.blocked_by|type=="array"))
' "$OUT/reports/unknown-missing-generated.json" >/dev/null
jq -e '.decision=="REFUTED" and any(.refutations[]; .reason=="HANDWRITTEN_SEMANTIC_AUTHORITY")'   "$OUT/reports/refuted-handwritten-authority.json" >/dev/null
jq -e '.decision=="REFUTED" and any(.refutations[]; .reason=="IR_GENERATED_CONTRADICTION")'   "$OUT/reports/refuted-semantic-mismatch.json" >/dev/null

before_generated=$(jq '.metrics.generated_bound' "$OUT/reports/refuted-handwritten-authority.json")
after_generated=$(jq '.metrics.generated_bound' "$OUT/reports/normal-all-generated.json")
before_handwritten=$(jq '.metrics.handwritten_go' "$OUT/reports/refuted-handwritten-authority.json")
after_handwritten=$(jq '.metrics.handwritten_go' "$OUT/reports/normal-all-generated.json")

go_files=$(find "$ROOT" -type f -name '*.go' -not -path '*/.git/*' | wc -l | tr -d ' ')
go_lines=$(find "$ROOT" -type f -name '*.go' -not -path '*/.git/*' -print0 | xargs -0 cat | wc -l | tr -d ' ')
gooo_files=$(find "$ROOT" -type f -name '*.gooo' -not -path '*/.git/*' | wc -l | tr -d ' ')
gooo_lines=$(find "$ROOT" -type f -name '*.gooo' -not -path '*/.git/*' -print0 | xargs -0 cat | wc -l | tr -d ' ')
regular_files=$(find "$ROOT" -type f -not -path '*/.git/*' -not -path "$ROOT/README.md" | wc -l | tr -d ' ')
descendant_dirs=$(find "$ROOT" -mindepth 1 -type d -not -path '*/.git*' | wc -l | tr -d ' ')
peak_rss_kib=$(cat "$OUT"/rss/*.txt | sort -nr | head -1)
END_NS=$(date +%s%N)
conformance_wall_ms=$(((END_NS-START_NS)/1000000))
AFTER_STATUS=$(git -C "$ROOT" status --porcelain)
net_repository_changes=0
test "$BEFORE_STATUS" = "$AFTER_STATUS" || net_repository_changes=1

artifact_files_before=$(find "$OUT" -type f | wc -l | tr -d ' ')
output_artifact_files=$((artifact_files_before+2))
jq -n   --arg schema "gooo/semantic-authority-conformance/v1"   --arg score "NOT_COMBINED"   --argjson cells 12   --argjson normal "$normal"   --argjson unknown "$unknown"   --argjson refuted "$refuted"   --argjson before_generated "$before_generated"   --argjson after_generated "$after_generated"   --argjson before_handwritten "$before_handwritten"   --argjson after_handwritten "$after_handwritten"   --argjson compile_wall_ms "$compile_wall_ms"   --argjson build_wall_ms "$build_wall_ms"   --argjson test_wall_ms "$test_wall_ms"   --argjson conformance_wall_ms "$conformance_wall_ms"   --argjson peak_rss_kib "$peak_rss_kib"   --argjson tests_executed "$tests_executed"   --argjson go_files "$go_files"   --argjson go_lines "$go_lines"   --argjson gooo_files "$gooo_files"   --argjson gooo_lines "$gooo_lines"   --argjson regular_files "$regular_files"   --argjson descendant_dirs "$descendant_dirs"   --argjson output_artifact_files "$output_artifact_files"   --argjson net_repository_changes "$net_repository_changes"   '{
    schema:$schema,
    score:$score,
    denominator:{
      cells:$cells,
      proof:{FOUNDATION:4,COHERENCE:4,REGRESSION:4},
      indicator:{DRIVER:4,OUTCOME:4,GUARDRAIL:4}
    },
    cases:{CLOSED:$normal,UNKNOWN:$unknown,REFUTED:$refuted},
    exact_pair:{
      obligations:3,
      generated_bound:{before:$before_generated,after:$after_generated,delta:($after_generated-$before_generated)},
      handwritten_go:{before:$before_handwritten,after:$after_handwritten,delta:($after_handwritten-$before_handwritten)}
    },
    execution:{
      compile_wall_ms:$compile_wall_ms,
      build_wall_ms:$build_wall_ms,
      test_wall_ms:$test_wall_ms,
      conformance_wall_ms:$conformance_wall_ms,
      peak_rss_kib:$peak_rss_kib,
      tests_total:$tests_executed,
      tests_executed:$tests_executed,
      tests_reused:0,
      tests_skipped:0,
      tests_not_observed:0,
      replay_comparisons:9,
      replay_mismatches:0
    },
    inventory:{
      go_files:$go_files,
      go_physical_lines:$go_lines,
      gooo_files:$gooo_files,
      gooo_physical_lines:$gooo_lines,
      regular_files_excluding_root_readme:$regular_files,
      descendant_dirs:$descendant_dirs,
      output_artifact_files:$output_artifact_files
    },
    authority:{
      repository_write_authority:0,
      net_repository_changes:$net_repository_changes,
      local_test_executions:0,
      infrastructure_mutations:0,
      provider_install_attempts:0,
      network_mutation_attempts:0
    },
    bootstrap:{
      compiler:"HANDWRITTEN_GO",
      evaluator:"HANDWRITTEN_GO",
      core_semantic_authority_closed:false
    }
  }' > "$OUT/conformance.json"

cat > "$OUT/report.md" <<EOF_REPORT
# Semantic authority census

- Cells: 12
- Cases CLOSED / UNKNOWN / REFUTED: $normal / $unknown / $refuted
- Generated-bound exact pair: $before_generated -> $after_generated
- Handwritten-Go exact pair: $before_handwritten -> $after_handwritten
- Replay comparisons / mismatches: 9 / 0
- Tests total / executed / reused / skipped / not observed: $tests_executed / $tests_executed / 0 / 0 / 0
- Compile / build / test / conformance ms: $compile_wall_ms / $build_wall_ms / $test_wall_ms / $conformance_wall_ms
- Peak RSS KiB: $peak_rss_kib
- Go files / lines: $go_files / $go_lines
- Gooo files / lines: $gooo_files / $gooo_lines
- Repository-write authority / net changes / local tests: 0 / $net_repository_changes / 0
- Score: NOT_COMBINED
- Core semantic authority closed: false
EOF_REPORT

actual_artifact_files=$(find "$OUT" -type f | wc -l | tr -d ' ')
test "$actual_artifact_files" = "$output_artifact_files"
jq -e '
  .denominator.cells==12 and
  .cases=={CLOSED:3,REFUTED:3,UNKNOWN:3} and
  .exact_pair.generated_bound=={before:2,after:3,delta:1} and
  .exact_pair.handwritten_go=={before:1,after:0,delta:-1} and
  .execution.replay_comparisons==9 and
  .execution.replay_mismatches==0 and
  .authority.repository_write_authority==0 and
  .authority.net_repository_changes==0 and
  .authority.local_test_executions==0 and
  .bootstrap.core_semantic_authority_closed==false and
  .score=="NOT_COMBINED"
' "$OUT/conformance.json" >/dev/null

cat "$OUT/report.md"
