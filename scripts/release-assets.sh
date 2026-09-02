#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 8 ]]; then
  echo "usage: release-assets.sh ROOT EVIDENCE_DIR PACKAGE TAG COMMIT MAIN_RUN_ID MAIN_ARTIFACT_ID MAIN_ARTIFACT_DIGEST" >&2
  exit 64
fi

ROOT=$1
EVIDENCE_DIR=$2
PACKAGE=$3
TAG=$4
COMMIT=$5
MAIN_RUN_ID=$6
MAIN_ARTIFACT_ID=$7
MAIN_ARTIFACT_DIGEST=$8

mkdir -p "$PACKAGE"
git -C "$ROOT" archive --format=tar.gz --prefix="gooo-semantic-authority-census-$TAG/" "$COMMIT" > "$PACKAGE/source-$TAG.tar.gz"
tar -czf "$PACKAGE/evidence-$TAG.tar.gz" -C "$EVIDENCE_DIR" .

source_digest="sha256:$(sha256sum "$PACKAGE/source-$TAG.tar.gz" | awk '{print $1}')"
evidence_digest="sha256:$(sha256sum "$PACKAGE/evidence-$TAG.tar.gz" | awk '{print $1}')"
jq -S -n \
  --arg schema 'gooo/self-description-boundary/release-manifest/v1' \
  --arg tag "$TAG" \
  --arg commit "$COMMIT" \
  --arg main_run "$MAIN_RUN_ID" \
  --arg main_artifact "$MAIN_ARTIFACT_ID" \
  --arg main_artifact_digest "$MAIN_ARTIFACT_DIGEST" \
  --arg source_digest "$source_digest" \
  --arg evidence_digest "$evidence_digest" \
  '{schema:$schema,tag:$tag,commit_sha:$commit,main_ci:{run_id:($main_run|tonumber),artifact_id:($main_artifact|tonumber),artifact_digest:$main_artifact_digest},assets:{source:{name:("source-"+$tag+".tar.gz"),digest:$source_digest},evidence:{name:("evidence-"+$tag+".tar.gz"),digest:$evidence_digest}},immutability:{draft_first:true,delete_or_reuse:false},claims:{aggregate_score:"NOT_EMITTED",percentage:"NOT_EMITTED",self_hosting:"NOT_EMITTED",improvement:"NOT_EMITTED"}}' \
  > "$PACKAGE/release-manifest-$TAG.json"

(cd "$PACKAGE" && sha256sum "source-$TAG.tar.gz" "evidence-$TAG.tar.gz" "release-manifest-$TAG.json" > SHA256SUMS)
