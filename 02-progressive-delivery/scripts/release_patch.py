#!/usr/bin/env python3
"""Build one JSON patch for a release: image tag and/or env changes.

stdin:  current Rollout as JSON (kubectl get rollout notes -o json)
args:   <image-tag> [KEY=VALUE ...] [KEY- ...]   (KEY- removes a variable, like kubectl)
stdout: JSON patch (an empty list when nothing changes)
stderr: human-readable summary of what changes

One patch = one pod template change = one revision. Changing image and env in
two separate commands would create two revisions back to back.
"""
import json
import sys

tag, pairs = sys.argv[1], sys.argv[2:]
rollout = json.load(sys.stdin)
container = rollout["spec"]["template"]["spec"]["containers"][0]

ops = []
old_image = container["image"]
new_image = old_image.rsplit(":", 1)[0] + ":" + tag
if new_image != old_image:
    ops.append({"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": new_image})
    print(f"   image  {old_image} -> {new_image}", file=sys.stderr)

env = [dict(e) for e in container.get("env", [])]
changed = False
for pair in pairs:
    if pair.endswith("-") and "=" not in pair:
        key = pair[:-1]
        if any(e["name"] == key for e in env):
            env = [e for e in env if e["name"] != key]
            print(f"   env    {key}: removed", file=sys.stderr)
            changed = True
        continue
    key, _, value = pair.partition("=")
    for e in env:
        if e["name"] == key:
            if e.get("value") != value:
                print(f"   env    {key}: {e.get('value')} -> {value}", file=sys.stderr)
                e["value"] = value
                changed = True
            break
    else:
        print(f"   env    {key}: (unset) -> {value}", file=sys.stderr)
        env.append({"name": key, "value": value})
        changed = True
if changed:
    ops.append({"op": "replace", "path": "/spec/template/spec/containers/0/env", "value": env})

json.dump(ops, sys.stdout)
