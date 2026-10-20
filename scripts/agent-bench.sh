#!/bin/bash
# Let go2lean's M2 agent write the specs (no hand-written ones installed)
# for the packages of the benchmark, run them, and list which benchmark
# findings its confirmed counterexamples hit.
# usage: scripts/agent-bench.sh <go2lean checkout> <tingly-box checkout> <workspace>
#   needs ANTHROPIC_API_KEY; AGENT_FLAGS passes more agent flags, e.g.
#   AGENT_FLAGS="-base-url https://api.minimaxi.com/anthropic -model MiniMax-M3 -thinking=false"
set -e
here=$(cd "$(dirname "$0")/.." && pwd)
G=$1; H=$2; WS=$3
# The agent writes its own go2lean_specs_test.go files: remove ours.
find "$H/internal/protocol" -name go2lean_specs_test.go -delete
bin=$(mktemp -d)/go2lean
(cd "$G" && go build -o "$bin" ./cmd/go2lean)
mkdir -p "$WS"
for p in request stream nonstream; do
  (cd "$G" && "$bin" agent -deps -module "$H" -workspace "$WS/$p" -guidance "$here/guidance/bug-classes.md" \
    $AGENT_FLAGS ./internal/protocol/$p) > "$WS/$p.log" 2>&1 || echo "agent $p failed" >> "$WS/$p.log"
done
echo "confirmed by the agent's specs:"
for p in request stream nonstream; do
  grep -h " confirmed " "$WS/$p.log" | sed "s/^/  [$p] /" | cut -c1-200
done
