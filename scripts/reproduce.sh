#!/bin/bash
# Reproduce every finding against tingly-box with plain `go test` (no
# go2lean, no Lean): copy the tests of repro/ into a checkout, run them,
# remove them again, and say per finding whether it still reproduces.
#
# usage: scripts/reproduce.sh [tingly-box checkout] [ref]
#   no checkout: clone https://github.com/tingly-dev/tingly-box into a
#   temporary directory; ref (a commit, tag or branch) is checked out first
#   when given. Needs Go and git.
set -e
here=$(cd "$(dirname "$0")/.." && pwd)
dir=${1:-}
ref=${2:-}
if [ -z "$dir" ]; then
  dir=$(mktemp -d)/tingly-box
  git clone -q --recurse-submodules https://github.com/tingly-dev/tingly-box "$dir"
fi
dir=$(cd "$dir" && pwd)
if [ -n "$ref" ]; then
  git -C "$dir" fetch -q origin "$ref" 2>/dev/null || true
  git -C "$dir" checkout -q --detach "$ref"
fi
# The SDKs are vendored as submodules (libs/).
git -C "$dir" submodule update -q --init --recursive
echo "tingly-box $(git -C "$dir" rev-parse --short HEAD) in $dir"

installed=()
cleanup() { for f in "${installed[@]}"; do rm -f "$f"; done; }
trap cleanup EXIT
pkgs=()
while read -r f; do
  rel=${f#"$here/repro/"}
  cp "$f" "$dir/$rel"
  installed+=("$dir/$rel")
  pkgs+=("./$(dirname "$rel")")
done < <(find "$here/repro" -name '*_test.go')

out=$(cd "$dir" && go test -count=1 -v -run '^TestGo2LeanRepro_' "${pkgs[@]}" 2>&1) || true

# Which finding each test reproduces.
declare -A finding=(
  [ToolCallIDsPairAfterConversion]="issue-1924-tool-call-ids (tracked #1924)"
  [SystemRoleMessageKeptOnChatPath]="issue-1921-system-messages, Chat path (tracked #1921)"
  [SystemRoleMessageKeptOnResponsesPath]="issue-1921-system-messages, Responses path (tracked #1921)"
  [ParallelToolCallsGetDistinctIndexes]="issue-1923-stream-indexes, parallel calls (tracked #1923)"
  [MaxTokensFinishesWithLength]="issue-1923-stream-indexes, max_tokens (tracked #1923)"
  [ThinkingReachesReasoningContent]="reasoning-content-nonstream (new)"
  [StreamedRefusalReachesChatClient]="responses-chat-stream-refusal (new)"
)
status=0
printf '\n%-36s  %-11s  %s\n' test result finding
for t in "${!finding[@]}"; do
  if grep -q -- "--- FAIL: TestGo2LeanRepro_$t " <<<"$out"; then r=reproduces
  elif grep -q -- "--- PASS: TestGo2LeanRepro_$t " <<<"$out"; then r=fixed
  else r=not-run; status=1
  fi
  printf '%-36s  %-11s  %s\n' "$t" "$r" "${finding[$t]}"
done | sort -k3
if grep -q -E '^(# |FAIL\s.*\[build failed\])' <<<"$out"; then
  echo; echo "build output:"; grep -E -A3 '^# |build failed' <<<"$out" | head -30
  status=1
fi
echo
grep -E '^\s+zz_go2lean_repro_test.go:[0-9]+:' <<<"$out" || true
exit $status
