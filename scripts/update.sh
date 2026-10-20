#!/bin/bash
# After tingly-box moves: check out a ref, install the specs, check they
# still compile, hunt every spec, verify the counterexamples with go test,
# and report which known findings still reproduce.
# usage: update.sh <go2lean checkout> <tingly-box checkout (writable)> <ref> <outdir>
set -e
here=$(cd "$(dirname "$0")/.." && pwd)
G=$(cd "$1" && pwd); H=$(cd "$2" && pwd); REF=$3; OUT=$4
mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd)
git -C "$H" fetch -q origin
git -C "$H" checkout -q --detach "$REF"
tb=$(git -C "$H" rev-parse --short HEAD); g2l=$(git -C "$G" rev-parse --short HEAD)
"$here/scripts/install-specs.sh" "$H" > /dev/null
if ! (cd "$H" && go vet ./internal/protocol/... > "$OUT/vet.log" 2>&1); then
  echo "specs no longer compile against tingly-box $tb: see $OUT/vet.log" >&2
  exit 1
fi
"$here/scripts/round.sh" "$G" "$H" "$OUT/ws" "_spec" "$OUT/round.log"
# Report: every spec property, and each known finding.
python3 - "$OUT" "$tb" "$g2l" "$here/findings/index.tsv" <<'PY'
import json, os, sys, re
out, tb, g2l, index = sys.argv[1:5]
ws = os.path.join(out, "ws")
verdicts = {}
for f in os.listdir(os.path.join(ws, "verdicts")) if os.path.isdir(os.path.join(ws, "verdicts")) else []:
    v = json.load(open(os.path.join(ws, "verdicts", f)))
    verdicts[f[:-5]] = v.get("status")
lines = open(os.path.join(out, "round.log")).read().splitlines()
results = {}
for l in lines:
    m = re.match(r"\s+(p_\S+) random: (.*)", l)
    if m:
        results[m.group(1)] = m.group(2)
def spec_of(prop):
    return prop.rsplit("_spec", 1)[-1]
rows = []
for prop, res in sorted(results.items()):
    cx = [c for c in verdicts if c.startswith("cx" + prop[1:])]
    status = ",".join(verdicts[c] for c in cx) if cx else ("refuted, not verified" if "found" in res else "holds")
    rows.append((spec_of(prop), status, res[:120]))
findings = []
for l in open(index):
    if l.startswith("#") or not l.strip():
        continue
    page, specs = l.rstrip("\n").split("\t")
    names = [s.lower().removeprefix("spec") for s in specs.split()]
    hit = [n for n in names if any(r[0] == n and "confirmed" in r[1] for r in rows)]
    findings.append((page, "still reproduces" if hit else "not reproduced"))
with open(os.path.join(out, "report.md"), "w") as w:
    w.write(f"# tingly-box {tb} with go2lean {g2l}\n\n## Known findings\n\n| finding | status |\n|---|---|\n")
    for p, s in findings:
        w.write(f"| {p} | {s} |\n")
    w.write("\n## Specs\n\n| spec | verdict | engine |\n|---|---|---|\n")
    for s, st, res in rows:
        w.write(f"| {s} | {st} | {res} |\n")
print(open(os.path.join(out, "report.md")).read())
PY
