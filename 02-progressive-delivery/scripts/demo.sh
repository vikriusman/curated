#!/usr/bin/env bash
# Canary release on Envoy Gateway, driven by Argo Rollouts, with two gates:
#   readiness       a canary that is not Ready never receives weight (progressDeadlineAbort)
#   Loki analysis   after each weight step, the canary's error rate from its request logs
#
#   1. good release v1.1.0                    Ready, error rate 0     -> 5/25/50/100, promoted
#   2. v1.1.0 + wrong DATABASE_URL            never Ready             -> no weight, aborted
#   3. remove the bad override                                        -> back to v1.1.0
#   4. v1.2.0 + FAULT_RATE=0.3                Ready, 30% of API 500s  -> Loki analysis fails, aborted
set -euo pipefail
cd "$(dirname "$0")/.."

CTX="${KCTX:?set KCTX}"
K="kubectl $CTX -n notes"
AR="kubectl argo rollouts $CTX -n notes"
BASE="${BASE:?set BASE, e.g. http://<server-ip>:8080}"

phase()   { $AR status notes --timeout 1s 2>/dev/null | head -1 || true; }
weights() { $K get httproute notes -o jsonpath='{range .spec.rules[0].backendRefs[*]}{.name}={.weight} {end}'; }

served() {
  # 40 real requests through Envoy: which version answered, and the status codes.
  for _ in $(seq 40); do curl -s -w '\n%{http_code}\n' "$BASE/version"; done | python3 scripts/report.py served
  printf '   served /api/notes: '
  for _ in $(seq 40); do curl -s -o /dev/null -w '%{http_code}\n' "$BASE/api/notes"; done | sort | uniq -c | awk '{printf "HTTP %s x%s  ", $2, $1} END {print ""}'
}

canary_readyz() {
  # The newest notes pod's own /readyz, fetched from inside the cluster by pod IP
  # (works even when the pod is not Ready; the image itself has no shell or curl).
  local pod ip
  pod="$($K get pods -l app=notes --sort-by=.metadata.creationTimestamp -o name | tail -1)"
  ip="$($K get "$pod" -o jsonpath='{.status.podIP}')"
  printf '   %s /readyz: ' "${pod#pod/}"
  kubectl $CTX -n notes run "readyz-$RANDOM" --rm -i --restart=Never --quiet \
    --image=curlimages/curl:8.11.1 -- -s "http://$ip:8080/readyz" | python3 scripts/report.py readyz
}

analysis() { echo "   Loki analysis runs (error rate per measurement):"; $K get analysisrun -o json | python3 scripts/report.py analysis; }

wait_rollout() {
  # Poll until the rollout settles; print every change of phase or route weights.
  local last="" now w start=$SECONDS
  while :; do
    now="$(phase)"; w="$(weights)"
    [ "$now|$w" != "$last" ] && printf '   %3ss  %-60.60s  %s\n' "$((SECONDS-start))" "$now" "$w" && last="$now|$w"
    case "$now" in Healthy*|Degraded*) break ;; esac
    sleep 2
  done
}

echo "== 0. baseline"
$AR get rollout notes | sed -n '/^Images/p'
echo "   route: $(weights)"
served

echo
echo "== 1. good release v1.1.0"
scripts/release.sh v1.1.0
wait_rollout
analysis
served

echo
echo "== 2. v1.1.0 with a wrong DATABASE_URL (dependency unreachable)"
scripts/release.sh v1.1.0 'DATABASE_URL=postgres://notes@db-typo.notes.svc:5432/notes?sslmode=disable'
sleep 15
canary_readyz
wait_rollout
served

echo
echo "== 3. remove the bad override"
scripts/release.sh v1.1.0 DATABASE_URL-
wait_rollout
served

echo
echo "== 4. v1.2.0 with FAULT_RATE=0.3 (Ready, but 30% of API requests fail)"
scripts/release.sh v1.2.0 FAULT_RATE=0.3
sleep 15
canary_readyz
wait_rollout
analysis
served
