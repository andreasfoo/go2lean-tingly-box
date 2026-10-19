#!/bin/bash
# One segment of a resumable full pass over a tingly-box checkout: every
# package, every function, no_panic and the specs found in the tree.
# Rerun after an interruption; it resumes from the hunt records.
# usage: full-pass.sh <go2lean checkout> <tingly-box checkout> <workspace> <log>
G=$1; H=$2; WS=$3; LOG=$4
bin=$(mktemp -d)/go2lean
(cd "$G" && go build -o "$bin" ./cmd/go2lean) || exit 1
# The PoC builds grow the Go build cache by gigabytes.
if [ "$(du -sm "$(go env GOCACHE)" | cut -f1)" -gt 8000 ]; then go clean -cache; fi
start=$(date +%s)
cd "$G"
"$bin" run -resume -deps -jobs 4 -budget 10 -module "$H" -workspace "$WS" ./... >> "$LOG" 2>&1
echo "SWEEPDONE exit $? after $(( $(date +%s) - start ))s" >> "$LOG"
