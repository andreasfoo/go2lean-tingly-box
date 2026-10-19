#!/bin/bash
# Run go2lean on the protocol packages of a tingly-box checkout with the
# specs installed, hunt the properties whose id matches a pattern, and
# verify the counterexamples with go test.
# usage: round.sh <go2lean checkout> <tingly-box checkout> <workspace> <grep pattern> <log>
set -e
G=$1; H=$2; WS=$3; PAT=$4; LOG=$5
bin=$(mktemp -d)/go2lean
(cd "$G" && go build -o "$bin" ./cmd/go2lean)
: > "$LOG"
rm -rf "$WS"
cd "$G"
"$bin" compile -deps -module "$H" -workspace "$WS" ./ai ./internal/protocol ./internal/protocol/wire \
  ./internal/protocol/stream ./internal/protocol/request ./internal/protocol/usage \
  ./internal/protocol/nonstream ./internal/protocol/sse > "$LOG.compile" 2>&1 || { echo "compile failed" >> "$LOG"; exit 1; }
for p in stream request; do
  "$bin" props -module "$H" -workspace "$WS" -package "github.com/tingly-dev/tingly-box/internal/protocol/$p" > /dev/null 2>&1
done
ls "$WS/properties" | grep -E "$PAT" | sed 's/\.json$//' > "$LOG.ids"
while read -r id; do
  "$bin" hunt -module "$H" -workspace "$WS" -property "$id" -budget 180 -maxlen 6 >> "$LOG" 2>&1
done < "$LOG.ids"
cxs=$(ls "$WS/counterexamples" 2>/dev/null | sed 's/\.json$//')
if [ -n "$cxs" ]; then
  "$bin" poc -module "$H" -workspace "$WS" -counterexample $cxs >> "$LOG" 2>&1
fi
echo ROUNDDONE >> "$LOG"
