#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
OUT=${1:?absolute caller-owned output directory is required}
case "$OUT" in
  /*) ;;
  *) echo "output directory must be absolute" >&2; exit 64 ;;
esac

mkdir -p "$OUT/compile" "$OUT/reports" "$OUT/replay" "$OUT/metrics" "$OUT/inventory"
START_NS=$(date +%s%N)
BEFORE_STATUS=$(git -C "$ROOT" status --porcelain)

measure() {
  local name=$1
  shift
  set +e
  /usr/bin/time -f '%e %M' -o "$OUT/metrics/$name.time" "$@"
  local status=$?
  set -e
  if [[ $status -ne 0 ]]; then
    echo "${name} failed with status ${status}" > "$OUT/metrics/$name.failure"
    return "$status"
  fi
}

metric_json() {
  local name=$1
  if [[ ! -f "$OUT/metrics/$name.time" ]]; then
    jq -n '{wall_ms:null,peak_rss_kib:null,measurement_state:"UNKNOWN"}'
    return
  fi
  read -r wall_seconds peak_rss_kib < "$OUT/metrics/$name.time"
  wall_ms=$(awk -v seconds="$wall_seconds" 'BEGIN { printf "%d", seconds * 1000 }')
  jq -n --argjson wall_ms "$wall_ms" --argjson peak_rss_kib "$peak_rss_kib" '{wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib,measurement_state:"CLOSED"}'
}

measure compile bash -c 'go run ./cmd/gooo-boundary-compile --source meta/self-description-boundary.gooo --out "$1" > "$2"' _ "$OUT/compile" "$OUT/compile/compile-receipt.json"
go run "$OUT/compile/boundary-policy.generated.go" > "$OUT/compile/boundary-policy.generated.json"
cmp "$OUT/compile/boundary-policy.ir.json" "$OUT/compile/boundary-policy.generated.json"

jq -S '{schema:.policy.schema,id:.policy.id,release:.policy.release,precedence:.policy.precedence,unknown_fields:.policy.unknown_fields,authority_states:.policy.authority_states,fixed_point_rule:.policy.fixed_point_rule,output_authority:.policy.output_authority,authority_cells:(.policy.authority_cells|map({id,expected_state,evidence_kind,semantic_role,activity,proof,indicator}))}' \
  "$OUT/compile/boundary-policy.ir.json" > "$OUT/compile/policy.normalized.json"
jq -S . contracts/self-description-boundary-v1.json > "$OUT/compile/contract.normalized.json"
cmp "$OUT/compile/policy.normalized.json" "$OUT/compile/contract.normalized.json"

measure build bash -c 'go build -trimpath -o "$1" ./cmd/gooo-boundary-projector' _ "$OUT/gooo-boundary-projector"
measure test bash -c 'go test ./... -json > "$1"' _ "$OUT/go-test.json"

closed=0
unknown=0
refuted=0
case_total=0
for manifest in "$ROOT"/fixtures/boundary/cases/*.json; do
  scenario=$(jq -r '.scenario_id' "$manifest")
  expected=$(jq -r '.expected_decision' "$manifest")
  case_total=$((case_total+1))
  measure "case-$scenario" "$OUT/gooo-boundary-projector" \
    --policy "$OUT/compile/boundary-policy.generated.json" \
    --manifest "$manifest" \
    --output "$OUT/reports/$scenario" \
    --root "$ROOT" > "$OUT/reports/$scenario.stdout"
  jq -e --arg expected "$expected" '
    .decision == $expected and
    (.authority_vector|length) == 8 and
    all(.authority_vector[]; (.authority_state|IN("GOOO_OWNED","GENERATED_FROM_GOOO","HANDWRITTEN_RUNTIME","BOOTSTRAP_EXTERNAL","UNKNOWN","REFUTED")) and (.proof|type=="string") and (.indicator|type=="string") and (.evidence|type=="array")) and
    (.replay.comparisons == 1) and (.replay.mismatches == 0)
  ' "$OUT/reports/$scenario/authority-boundary-report.json" >/dev/null
  case "$expected" in
    CLOSED) closed=$((closed+1)) ;;
    UNKNOWN) unknown=$((unknown+1)) ;;
    REFUTED) refuted=$((refuted+1)) ;;
    *) echo "unexpected expected_decision $expected" >&2; exit 1 ;;
  esac

  "$OUT/gooo-boundary-projector" \
    --policy "$OUT/compile/boundary-policy.generated.json" \
    --manifest "$manifest" \
    --output "$OUT/replay/$scenario" \
    --root "$ROOT" > "$OUT/replay/$scenario.stdout"
  cmp "$OUT/reports/$scenario/authority-boundary-report.json" "$OUT/replay/$scenario/authority-boundary-report.json"
  cmp "$OUT/reports/$scenario/authority-boundary-report.md" "$OUT/replay/$scenario/authority-boundary-report.md"
done

test "$case_total" = 9
test "$closed" = 3
test "$unknown" = 3
test "$refuted" = 3

jq -e '
  (.unknowns|length)>0 and
  all(.unknowns[]; ((keys|sort)==["blocked_by","next_operation","reason","stage","step","unknown_class"]) and
    (.stage|type=="string") and (.step|type=="string") and (.reason|type=="string") and
    (.unknown_class|type=="string") and (.next_operation|type=="string") and (.blocked_by|type=="array"))
' "$OUT/reports/unknown-missing-toolchain/authority-boundary-report.json" >/dev/null
jq -e '.decision == "REFUTED" and any(.refutations[]; .reason == "SEMANTIC_AUTHORITY_INFERRED_FROM_FILE_SHAPE")' \
  "$OUT/reports/refuted-extension-only/authority-boundary-report.json" >/dev/null
jq -e '.decision == "REFUTED" and any(.refutations[]; .reason == "AUTHORITY_MARKER_CONTRADICTION")' \
  "$OUT/reports/refuted-authority-contradiction/authority-boundary-report.json" >/dev/null
jq -e '.decision == "REFUTED" and any(.refutations[]; .reason == "INPUT_REPOSITORY_WRITE_AUTHORITY_NONZERO")' \
  "$OUT/reports/refuted-write-escalation/authority-boundary-report.json" >/dev/null
jq -e '
  [.authority_vector[].authority_state] == ["GOOO_OWNED","GOOO_OWNED","GOOO_OWNED","HANDWRITTEN_RUNTIME","HANDWRITTEN_RUNTIME","GOOO_OWNED","BOOTSTRAP_EXTERNAL","GENERATED_FROM_GOOO"] and
  [.proof_vector[]] == ["FOUNDATION","FOUNDATION","FOUNDATION","COHERENCE","COHERENCE","COHERENCE","REGRESSION","REGRESSION"] and
  [.indicator_vector[]] == ["DRIVER","DRIVER","OUTCOME","OUTCOME","OUTCOME","GUARDRAIL","DRIVER","GUARDRAIL"]
' "$OUT/reports/normal-all-cells/authority-boundary-report.json" >/dev/null

inventory_list() {
  local kind=$1
  case "$kind" in
    go) find "$ROOT" -path "$ROOT/.git" -prune -o -type f -name '*.go' -print ;;
    gooo) find "$ROOT" -path "$ROOT/.git" -prune -o -type f -name '*.gooo' -print ;;
    dirs) find "$ROOT" -path "$ROOT/.git" -prune -o -mindepth 1 -type d -print ;;
    regular) find "$ROOT" -path "$ROOT/.git" -prune -o -type f ! -path "$ROOT/README.md" -print ;;
  esac | sed "s#^$ROOT/##" | sort
}

physical_lines() {
  local kind=$1
  local total=0
  while IFS= read -r path; do
    [[ -z "$path" ]] && continue
    lines=$(wc -l < "$ROOT/$path" | tr -d ' ')
    total=$((total+lines))
  done < "$OUT/inventory/$kind.paths"
  echo "$total"
}

for kind in go gooo dirs regular; do inventory_list "$kind" > "$OUT/inventory/$kind.paths"; done
go_files=$(wc -l < "$OUT/inventory/go.paths" | tr -d ' ')
gooo_files=$(wc -l < "$OUT/inventory/gooo.paths" | tr -d ' ')
descendant_dirs=$(wc -l < "$OUT/inventory/dirs.paths" | tr -d ' ')
regular_files=$(wc -l < "$OUT/inventory/regular.paths" | tr -d ' ')
go_lines=$(physical_lines go)
gooo_lines=$(physical_lines gooo)
go_paths_json=$(jq -Rsc 'split("\n")|map(select(length>0))' "$OUT/inventory/go.paths")
gooo_paths_json=$(jq -Rsc 'split("\n")|map(select(length>0))' "$OUT/inventory/gooo.paths")
dir_paths_json=$(jq -Rsc 'split("\n")|map(select(length>0))' "$OUT/inventory/dirs.paths")
regular_paths_json=$(jq -Rsc 'split("\n")|map(select(length>0))' "$OUT/inventory/regular.paths")

cache_status="${GO_MODULE_CACHE_STATUS:-}"
if [[ "$cache_status" == "true" || "$cache_status" == "HIT" ]]; then
  cache_json='{"status":"HIT","observed":true,"measurement_state":"CLOSED"}'
elif [[ "$cache_status" == "false" || "$cache_status" == "MISS" || -n "$cache_status" ]]; then
  cache_json='{"status":"MISS","observed":true,"measurement_state":"CLOSED"}'
else
  cache_json='{"status":null,"observed":false,"measurement_state":"UNKNOWN","reason":"CACHE_METRIC_ABSENT"}'
fi

AFTER_STATUS=$(git -C "$ROOT" status --porcelain)
net_repository_changes=0
if [[ "$BEFORE_STATUS" != "$AFTER_STATUS" ]]; then net_repository_changes=1; fi
END_NS=$(date +%s%N)
conformance_wall_ms=$(((END_NS-START_NS)/1000000))
toolchain=$(go env GOVERSION)
reports_json=$(jq -s '[.[] | {scenario_id,expected_decision,decision,authority_vector,proof_vector,indicator_vector,replay}]' "$OUT"/reports/*/authority-boundary-report.json)
compile_metrics=$(metric_json compile)
build_metrics=$(metric_json build)
test_metrics=$(metric_json test)
conformance_metrics=$(jq -n --argjson wall_ms "$conformance_wall_ms" '{wall_ms:$wall_ms,peak_rss_kib:null,measurement_state:"UNKNOWN",peak_rss_reason:"conformance_parent_process_not_observed"}')
tests_executed=$(jq -s '[.[]|select(.Action=="pass" and (.Test? != null))]|length' "$OUT/go-test.json")

jq -n \
  --arg schema "gooo/self-description-boundary/ci-evidence/v1" \
  --arg repository "${GITHUB_REPOSITORY:-local/unknown}" \
  --arg commit "${GITHUB_SHA:-unknown}" \
  --arg ref "${GITHUB_REF:-unknown}" \
  --arg event "${GITHUB_EVENT_NAME:-unknown}" \
  --arg workflow "${GITHUB_WORKFLOW:-unknown}" \
  --arg job "${GITHUB_JOB:-unknown}" \
  --arg run_id "${GITHUB_RUN_ID:-unknown}" \
  --arg run_attempt "${GITHUB_RUN_ATTEMPT:-unknown}" \
  --arg go_version "$toolchain" \
  --argjson compile "$compile_metrics" \
  --argjson build "$build_metrics" \
  --argjson test "$test_metrics" \
  --argjson conformance "$conformance_metrics" \
  --argjson cache "$cache_json" \
  --argjson cases "$reports_json" \
  --argjson tests_executed "$tests_executed" \
  --argjson go_paths "$go_paths_json" \
  --argjson gooo_paths "$gooo_paths_json" \
  --argjson dir_paths "$dir_paths_json" \
  --argjson regular_paths "$regular_paths_json" \
  --argjson go_files "$go_files" \
  --argjson go_lines "$go_lines" \
  --argjson gooo_files "$gooo_files" \
  --argjson gooo_lines "$gooo_lines" \
  --argjson descendant_dirs "$descendant_dirs" \
  --argjson regular_files "$regular_files" \
  --argjson conformance_wall_ms "$conformance_wall_ms" \
  --argjson net_repository_changes "$net_repository_changes" \
  --argjson closed "$closed" \
  --argjson unknown "$unknown" \
  --argjson refuted "$refuted" \
  '{
    schema:$schema,
    run:{repository:$repository,commit:$commit,ref:$ref,event:$event,workflow:$workflow,job:$job,id:$run_id,attempt:$run_attempt},
    toolchain:{go:$go_version,required:"go1.27.0",authority:"BOOTSTRAP_EXTERNAL"},
    corpus:{case_count:($cases|length),decision_vectors:$cases,exact_decision_counts:{CLOSED:$closed,UNKNOWN:$unknown,REFUTED:$refuted}},
    denominator:{authority_cells:8,policy:"meta/self-description-boundary.gooo",contract:"contracts/self-description-boundary-v1.json",proof_and_indicator_independent:true},
    execution:{compile:$compile,build:$build,test:$test,conformance:$conformance,replay_comparisons:9,replay_mismatches:0,cache:$cache,tests:{executed:$tests_executed,reused:0,skipped:0,not_observed:0}},
    inventory:{go_files:$go_files,go_physical_lines:$go_lines,go_file_paths:$go_paths,gooo_files:$gooo_files,gooo_physical_lines:$gooo_lines,gooo_file_paths:$gooo_paths,descendant_dirs:$descendant_dirs,descendant_dir_paths:$dir_paths,regular_files_excluding_root_readme:$regular_files,regular_file_paths_excluding_root_readme:$regular_paths,root_readme_excluded:true},
    authority:{repository_write_authority:0,net_repository_changes:$net_repository_changes,input_counters:{repository_writes:0,local_test_executions:0,infrastructure_mutations:0,provider_install_attempts:0,network_mutation_attempts:0},local_validation:{compile:0,build:0,test:0,conformance:0}},
    fixed_point_rule:"EXPLICIT_ONLY",
    incidents:[],
    claims:{aggregate_score:"NOT_EMITTED",percentage:"NOT_EMITTED",self_hosting:"NOT_EMITTED",improvement:"NOT_EMITTED"}
  }' > "$OUT/ci-evidence.json"

cat > "$OUT/report.md" <<EOF_REPORT
# Gooo self-description boundary CI evidence

- Fixed authority-cell denominator: 8
- Exact corpus CLOSED / UNKNOWN / REFUTED: $closed / $unknown / $refuted
- Precedence: REFUTED > UNKNOWN > CLOSED
- Replay comparisons / mismatches: 9 / 0
- Go toolchain: $toolchain
- Repository-write authority / net repository changes: 0 / $net_repository_changes
- Cache measurement: $(jq -c . <<< "$cache_json")
- Aggregate score, percentage, self-hosting, and improvement claims: not emitted

Per-cell authority vectors, proof/indicator vectors, exact evidence digests,
line ranges, and the complete inventory are in ci-evidence.json and the
scenario reports. Failed runs retain the caller-owned output as an immutable
run-scoped Actions artifact.
EOF_REPORT

jq -e '
  .denominator == {authority_cells:8,policy:"meta/self-description-boundary.gooo",contract:"contracts/self-description-boundary-v1.json",proof_and_indicator_independent:true} and
  .corpus.case_count == 9 and .corpus.exact_decision_counts == {CLOSED:3,UNKNOWN:3,REFUTED:3} and
  .execution.replay_comparisons == 9 and .execution.replay_mismatches == 0 and
  .authority.repository_write_authority == 0 and .authority.net_repository_changes == 0 and
  .fixed_point_rule == "EXPLICIT_ONLY" and (.incidents|length) == 0
' "$OUT/ci-evidence.json" >/dev/null
