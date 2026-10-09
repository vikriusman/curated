#!/usr/bin/env bash
# Record what Argo Rollouts does around a release, every 2 s, until the old
# ReplicaSet has been scaled to 0: rollout phase, HTTPRoute weights, which
# ReplicaSet (pod-template-hash) each Service selects, and replicas per ReplicaSet.
#
#   KCTX="--kubeconfig ..." scripts/watch-promotion.sh
set -euo pipefail
CTX="${KCTX:?set KCTX}"
K="kubectl $CTX -n notes"
start=$SECONDS; last=""; settled=0
printf '%5s  %-28s %-12s %-12s %-12s %s\n' "t" "phase" "weights" "stable→" "canary→" "ReplicaSets (hash ready/desired)"
while :; do
  phase="$(kubectl argo rollouts $CTX -n notes status notes --timeout 1s 2>/dev/null | head -1 | cut -c1-28 || true)"
  w="$($K get httproute notes -o jsonpath='{.spec.rules[0].backendRefs[0].weight}/{.spec.rules[0].backendRefs[1].weight}')"
  s="$($K get svc notes-stable -o jsonpath='{.spec.selector.rollouts-pod-template-hash}')"
  c="$($K get svc notes-canary -o jsonpath='{.spec.selector.rollouts-pod-template-hash}')"
  rs="$($K get rs -l app=notes -o jsonpath='{range .items[?(@.spec.replicas>0)]}{.metadata.labels.rollouts-pod-template-hash}:{.status.readyReplicas}/{.spec.replicas} {end}')"
  line="$phase|$w|$s|$c|$rs"
  if [ "$line" != "$last" ]; then
    printf '%4ss  %-28s %-12s %-12s %-12s %s\n' "$((SECONDS-start))" "$phase" "$w" "$s" "$c" "$rs"
    last="$line"
  fi
  # stop once healthy and only one ReplicaSet still has replicas, plus a margin
  if [[ "$phase" == Healthy* ]] && [ "$(wc -w <<<"$rs")" -eq 1 ] && [ $((SECONDS-start)) -gt 30 ]; then
    settled=$((settled+1)); [ $settled -ge 3 ] && break
  fi
  sleep 2
done
