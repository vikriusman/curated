#!/usr/bin/env bash
# Ship a release = build artifact (image tag) + config (12-factor V).
#
#   scripts/release.sh v1.1.0                 # code release
#   scripts/release.sh v1.1.0 LOG_LEVEL=debug # config change (same image)
#   scripts/release.sh v1.1.0 LOG_LEVEL-      # remove a variable again
#
# 1. computes one patch for image + env (one revision, not two)
# 2. runs the schema migration with the new image (admin process, XII)
# 3. patches the Rollout; Argo Rollouts drives the canary from there
set -euo pipefail
cd "$(dirname "$0")/.."

TAG="${1:?usage: release.sh <image-tag> [KEY=VALUE ...] [KEY- ...]}"; shift
CTX="${KCTX:?set KCTX, e.g. --kubeconfig ../01-stateless-monolith/terraform/aws/kubeconfig}"
K="kubectl $CTX -n notes"

echo "== release $TAG ${*:-}"
PATCH="$($K get rollout notes -o json | python3 scripts/release_patch.py "$TAG" "$@")"
if [ "$PATCH" = "[]" ]; then
  echo "   no changes"
  exit 0
fi

$K delete job notes-migrate --ignore-not-found >/dev/null
sed "s#image: notes:TAG#image: notes:$TAG#" k8s/jobs/migrate.yaml | kubectl $CTX apply -f - >/dev/null
$K wait --for=condition=complete job/notes-migrate --timeout=120s >/dev/null
echo "   migrate: $($K logs job/notes-migrate | tail -1)"

$K patch rollout notes --type=json -p "$PATCH" >/dev/null
echo "   patched; Argo Rollouts takes over"
