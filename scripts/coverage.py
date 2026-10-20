#!/usr/bin/env python3
"""Function-level model coverage of a go2lean workspace, as Markdown.

usage: scripts/coverage.py <workspace or model/reports> <go2lean commit>

Reads the model reports (a workspace's models/gossa/*/report.json, or the
copies in model/reports/*.json) of the tingly-box
packages and prints, per package, how many functions go2lean translated
into Lean IR, then every function with its status:

  complete  translated; every call it makes is modelled
  partial   translated; some calls or values are havoc (unmodelled), the
            report lists them; a counterexample that reads one is checked
            by go test like any other
  rejected  not translated (the reason is listed); callers treat a call to
            it as havoc

Functions declared in _test.go files (specs, test helpers) and closures
(counted with the function that holds them) are left out.
"""
import glob
import json
import os
import sys
from collections import Counter

PREFIX = "github.com/tingly-dev/tingly-box/"

ws, rev = sys.argv[1], sys.argv[2]
pkgs = {}
files = glob.glob(os.path.join(ws, "models/gossa/*/report.json")) or glob.glob(os.path.join(ws, "*.json"))
for f in sorted(files):
    r = json.load(open(f))
    if not r["package"].startswith(PREFIX):
        continue
    rows = []
    for fn in r["funcs"]:
        issues = fn.get("issues") or []
        if fn.get("test_file") or fn.get("closure") or "$" in fn["name"]:
            continue
        if any(i.startswith("test helper:") for i in issues):
            continue  # a helper of a _test.go file (takes *testing.T)
        note = ""
        if fn["status"] == "rejected":
            note = issues[0] if issues else ""
        elif fn["status"] == "partial":
            note = f"{len(issues)} unmodelled site{'s' if len(issues) != 1 else ''}"
            if issues:
                note += ": " + issues[0]
        rows.append((fn["name"], fn["status"], note))
    pkgs[r["package"][len(PREFIX):]] = rows

total = Counter()
print(f"Model of go2lean `{rev}` (`compile -deps` of the protocol packages; the")
print("other tingly-box packages are compiled as far as those call into them).\n")
print("| package | functions | complete | partial | rejected | in Lean IR |")
print("|---|---|---|---|---|---|")
for pkg, rows in sorted(pkgs.items()):
    c = Counter(s for _, s, _ in rows)
    n = len(rows)
    total.update(c)
    total["n"] += n
    pct = 100 * (c["complete"] + c["partial"]) / n if n else 0
    print(f"| `{pkg}` | {n} | {c['complete']} | {c['partial']} | {c['rejected']} | {pct:.0f}% |")
n = total["n"]
pct = 100 * (total["complete"] + total["partial"]) / n if n else 0
print(f"| **all** | **{n}** | **{total['complete']}** | **{total['partial']}** | **{total['rejected']}** | **{pct:.0f}%** |")
print()
for pkg, rows in sorted(pkgs.items()):
    print(f"<details><summary><code>{pkg}</code>: {len(rows)} functions</summary>\n")
    print("| function | status | note |")
    print("|---|---|---|")
    for name, status, note in sorted(rows, key=lambda r: (r[1] != "rejected", r[1] != "partial", r[0])):
        note = note.replace("|", "\\|").replace("\n", " ")
        if len(note) > 140:
            note = note[:137] + "..."
        print(f"| `{name}` | {status} | {note} |")
    print("\n</details>\n")
