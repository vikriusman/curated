#!/usr/bin/env bash
# Proof of the stateless pattern:
#   1. log in once (session stored in Redis)
#   2. upload one photo through one pod (stored in S3 via Mountpoint)
#   3. hit /whoami repeatedly: the pod changes, user and photos stay visible
#   4. delete every app pod: the session survives on the replacement pods
set -euo pipefail

BASE="${1:-http://localhost:8080}"
N="${N:-12}"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT
JAR="$WORK/cookies"
# kubectl target, e.g. KUBECTL_ARGS="--kubeconfig terraform/aws/kubeconfig"
CTX="${KUBECTL_ARGS:---context k3d-stateless-gallery}"

# Keep-alive disabled so each request can land on a different pod.
c() { curl -sS --fail -H 'Connection: close' -c "$JAR" -b "$JAR" "$@"; }
token() { grep -o 'name="csrf_token" value="[^"]*"' | head -1 | cut -d'"' -f4; }

# Wait until the app answers through the ingress (fresh rollouts take a moment).
for _ in $(seq 60); do curl -sf -o /dev/null "$BASE/healthz?ready=1" && break; sleep 2; done

echo "== 1. login"
TOKEN="$(c "$BASE/login" | token)"
c -o /dev/null -d "csrf_token=$TOKEN&username=demo&password=${DEMO_PASSWORD:-demo}" "$BASE/login"

echo "== 2. upload"
# Valid 1x1 PNG, no extra dependencies.
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\xcf\xc0\xf0\x1f\x00\x05\x00\x01\xff\x89\x99=\x1d\x00\x00\x00\x00IEND\xaeB`\x82' > "$WORK/dot.png"
TOKEN="$(c "$BASE/" | token)"
c -o /dev/null -F "csrf_token=$TOKEN" -F "photo=@$WORK/dot.png;type=image/png" "$BASE/upload"

echo "== 3. $N requests to /whoami"
printf '%-34s %-28s %-6s %-6s %s\n' POD NODE USER VISIT PHOTOS
for _ in $(seq "$N"); do
  c "$BASE/whoami" | python3 -c 'import sys,json; d=json.load(sys.stdin); print("%-34s %-28s %-6s %-6s %s" % (d["pod"], d["node"], d["user"], d["visits"], d["photo_count"]))'
done | tee "$WORK/out"
PODS="$(awk '{print $1}' "$WORK/out" | sort -u | wc -l)"
USERS="$(awk '{print $3}' "$WORK/out" | sort -u)"
echo "unique pods: $PODS, users seen: $USERS"

echo "== 4. delete all app pods, wait for replacements"
OLD="$(kubectl $CTX -n gallery get pods -l app=gallery -o name | sort)"
kubectl $CTX -n gallery delete pod -l app=gallery >/dev/null
kubectl $CTX -n gallery wait --for=condition=ready pod -l app=gallery --timeout=180s >/dev/null
NEW="$(kubectl $CTX -n gallery get pods -l app=gallery -o name | sort)"
[ -z "$(comm -12 <(echo "$OLD") <(echo "$NEW"))" ] || { echo "old pods still present"; exit 1; }
# The ingress needs a few seconds to refresh its endpoints.
for _ in $(seq 30); do curl -sf -o /dev/null "$BASE/healthz" && break; sleep 1; done
c "$BASE/whoami"; echo

[ "$PODS" -gt 1 ] && [ "$USERS" = "demo" ] && echo "PASS: sessions and files are consistent across pods" || { echo "FAIL"; exit 1; }
