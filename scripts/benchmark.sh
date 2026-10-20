#!/bin/bash
# The benchmark of go2lean on bugs first found by hand: for every spec in
# benchmark.tsv, hunt it blind with go2lean, verify what it finds with go
# test, and print one row per case (found at which trial, verdict, or what
# the search reported). Run it after every go2lean change.
# usage: scripts/benchmark.sh <go2lean checkout> <tingly-box checkout> <workspace>
#   REUSE=1 hunts again in a workspace compiled by an earlier run.
set -e
here=$(cd "$(dirname "$0")/.." && pwd)
G=$1; H=$2; WS=$3
"$here/scripts/install-specs.sh" "$H" > /dev/null
pat=$(grep -v '^#' "$here/benchmark.tsv" | cut -f1 | tr 'A-Z' 'a-z' | paste -sd'|')
BUDGET=${BUDGET:-300} JOBS=${JOBS:-2} "$here/scripts/round.sh" "$G" "$H" "$WS" "_($pat)\.json\$" "$WS.log"
python3 - "$here/benchmark.tsv" "$WS" "$(git -C "$G" rev-parse --short HEAD)" <<'PY'
import glob, json, os, sys
tsv, ws, rev = sys.argv[1:]
print(f"go2lean {rev}\n")
print("| spec | finding | search | verdict |\n|---|---|---|---|")
hits = 0
cases = [l.rstrip("\n").split("\t") for l in open(tsv) if l.strip() and not l.startswith("#")]
for spec, page, note in cases:
    key = spec.lower()
    hunt = [f for f in glob.glob(f"{ws}/hunts/*.json") if f.endswith("_" + key + ".json")]
    summary = json.load(open(hunt[0]))["summary"] if hunt else "not hunted"
    search = summary.split(";")[0].replace("random: ", "")
    verdict = "-"
    for f in glob.glob(f"{ws}/verdicts/*.json"):
        if "_" + key + "_" in os.path.basename(f):
            verdict = json.load(open(f))["status"]
    if verdict == "confirmed":
        hits += 1
    if not search.startswith("found"):
        search = summary[:160]
    print(f"| {spec} ({note}) | {page} | {search} | {verdict} |")
print(f"\n{hits} of {len(cases)} confirmed")
PY
