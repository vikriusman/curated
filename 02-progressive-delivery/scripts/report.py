#!/usr/bin/env python3
"""Small formatters for demo.sh (stdin in, one human line out)."""
import collections
import json
import sys


def served():
    # stdin: alternating lines "<json body>" and "<http code>"
    lines = [l.strip() for l in sys.stdin if l.strip()]
    c = collections.Counter()
    for body, code in zip(lines[0::2], lines[1::2]):
        try:
            d = json.loads(body)
            c["HTTP %s %s" % (code, d["version"])] += 1
        except (ValueError, KeyError):
            c["HTTP %s" % code] += 1
    print("   served /version:", ", ".join("%s x%d" % kv for kv in c.most_common()))


def readyz():
    d = json.load(sys.stdin)
    checks = ", ".join("%s=%s" % (x["name"], "ok" if x["ok"] else "FAIL") for x in d.get("checks", []))
    print("%s %s  (%s)" % ("READY" if d["ready"] else "NOT READY", d.get("version", ""), checks))


def analysis():
    runs = sorted(json.load(sys.stdin)["items"], key=lambda r: r["metadata"]["creationTimestamp"])
    if not runs:
        print("   (no analysis run)")
        return
    for r in runs[-3:]:
        s = r.get("status", {})
        for m in s.get("metricResults", []):
            vals = []
            for x in m.get("measurements", []):
                v = (x.get("value") or "").strip('"')   # web provider returns a JSON string
                try:
                    vals.append("%.3f" % float(v))
                except ValueError:
                    vals.append(x.get("phase", "?"))
            print("   %-32s %-11s %s: %s" % (r["metadata"]["name"], s.get("phase", ""), m["name"], " ".join(vals)))


{"served": served, "readyz": readyz, "analysis": analysis}[sys.argv[1]]()
