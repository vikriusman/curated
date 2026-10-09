#!/usr/bin/env bash
# Re-export architecture.drawio.svg after editing it in draw.io.
# draw.io emits theme-adaptive colors (CSS light-dark()), which turn black text
# white in dark-mode viewers while the boxes stay light. This pins every color
# to its light value so the diagram reads the same everywhere.
set -euo pipefail
cd "$(dirname "$0")"
SVG=architecture.drawio.svg
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT

drawio -x -f xml --uncompressed -o "$TMP/src.drawio" "$SVG" --no-sandbox >/dev/null 2>&1
drawio -x -f svg -e --svg-theme light --embed-svg-fonts false -o "$SVG" "$TMP/src.drawio" --no-sandbox >/dev/null 2>&1

python3 - "$SVG" <<'PY'
import re, sys
p = sys.argv[1]; s = open(p).read()
s = re.sub(r' style="fill: light-dark\([^"]*\);"', '', s)
pat = re.compile(r'light-dark\(((?:[^(),]|\([^()]*\))+),\s*((?:[^(),]|\([^()]*\))+)\)')
s = pat.sub(lambda m: m.group(1).strip(), s)
s = re.sub(r'color-scheme:\s*[^;"]*;?', '', s)
s = s.replace('background: transparent; background-color: transparent', 'background: #ffffff; background-color: #ffffff', 1)
assert 'light-dark' not in s
open(p, 'w').write(s)
PY
echo "exported $SVG"
