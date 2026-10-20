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
  [EnvAuthTokenNotSentUpstream]="env-credentials-upstream, ANTHROPIC_AUTH_TOKEN (new)"
  [EnvOpenAIOrgNotSentUpstream]="env-credentials-upstream, OPENAI_ORG_ID (new)"
  [NullReasoningOpensNoThinkingBlock]="chat-anthropic-reasoning-null, stream (new)"
  [NonStreamThinkingIsPlainText]="chat-anthropic-reasoning-null, non-stream (new)"
  [DoneOnlyArgumentsReachChatClient]="responses-done-only-arguments, Chat client (new)"
  [DoneOnlyArgumentsReachAnthropicClient]="responses-done-only-arguments, Anthropic client (new)"
  [ChatRequestKeepsSamplingAndStop]="chat-request-params, sampling and stop (new)"
  [ChatRequestKeepsToolChoiceMode]="chat-request-params, tool_choice (new)"
  [NonStreamUsageExcludesCachedInput]="usage-nonstream-cached-input (new)"
  [ResponsesStreamUsageAfterFinishChunk]="responses-stream-trailing-usage, converter (new)"
  [GatewayResponsesStreamGetsUsage]="responses-stream-trailing-usage, gateway (new)"
  [ToolCallIndexAfterReasoningItem]="responses-chat-tool-call-index (new)"
  [ResponsesDeveloperMessageStaysDeveloper]="responses-developer-message-as-user, request (new)"
  [GatewayDeveloperRoleReachesChatUpstream]="responses-developer-message-as-user, gateway (new)"
  [AbnormalStopReasonsNotCompleted]="anthropic-abnormal-stop-reasons, non-stream (new)"
  [ContextWindowExceededStreamIncomplete]="anthropic-abnormal-stop-reasons, stream (new)"
  [StreamToolCallWithoutIDGetsID]="chat-tool-call-empty-id, stream (new)"
  [NonStreamToolCallWithoutIDGetsID]="chat-tool-call-empty-id, non-stream (new)"
  [GatewayToolCallWithoutIDGetsID]="chat-tool-call-empty-id, gateway (new)"
  [AnthropicRequestKeepsDeveloperAndTools]="chat-anthropic-request-coverage, developer and tools (new)"
  [AnthropicRequestKeepsSamplingAndStop]="chat-anthropic-request-coverage, sampling and stop (new)"
  [GatewayAnthropicUpstreamGetsChatParams]="chat-anthropic-request-coverage, gateway (new)"
  [ResponsesRequestKeepsTokensAndSampling]="chat-responses-request-params, request (new)"
  [GatewayResponsesUpstreamGetsChatParams]="chat-responses-request-params, gateway (new)"
  [GatewayErrorIsAnthropicShaped]="upstream-errors-not-translated, Anthropic client (new)"
  [GatewayErrorCodeReachesResponsesClient]="upstream-errors-not-translated, Responses client (new)"
  [AnthropicToChatToolSchemaKeepsKeywords]="anthropic-tool-schema-keywords, Chat provider (new)"
  [AnthropicToResponsesToolSchemaKeepsKeywords]="anthropic-tool-schema-keywords, Responses provider (new)"
  [ChatResponseFormatCarried]="structured-output-dropped, Chat client (new)"
  [ResponsesTextFormatCarried]="structured-output-dropped, Responses client (new)"
  [AnthropicOutputFormatCarried]="structured-output-dropped, Anthropic client (new)"
  [ReasoningItemNoSummaryOneOpenBlock]="responses-anthropic-stream-open-thinking, no summary (new)"
  [ReasoningItemWithSummaryOneOpenBlock]="responses-anthropic-stream-open-thinking, summary (new)"
  [ParameterlessToolChatStreamArgsAreJSON]="parameterless-tool-stream-arguments, Chat client (new)"
  [ParameterlessToolResponsesStreamArgsAreJSON]="parameterless-tool-stream-arguments, Responses client (new)"
  [AnthropicDocumentBlockCarried]="document-content-dropped, Anthropic client (new)"
  [ChatFilePartCarried]="document-content-dropped, Chat client (new)"
  [ResponsesRefusalReachesAnthropicClient]="responses-anthropic-nonstream-refusal (new)"
  [ResponsesAssistantTurnIsOneChatMessage]="responses-chat-split-assistant-turn (new)"
  [NonStreamReasoningSummaryIsThinking]="responses-anthropic-nonstream-reasoning-summary (new)"
  [ToolOutputImageReachesAnthropic]="responses-anthropic-tool-output-image (new)"
  [ChatToResponsesKeepsStrict]="chat-responses-tool-strict, Chat -> Responses (new)"
  [ResponsesToChatKeepsStrict]="chat-responses-tool-strict, Responses -> Chat (new)"
  [AccumulatorDeltaRepeatingCacheCreation]="usage-anthropic-delta-cache-creation, creation repeated (new)"
  [AccumulatorDeltaRepeatingInput]="usage-anthropic-delta-cache-creation, input repeated (new)"
  [ParseAnthropicKeepsAllTextBlocks]="sse-parse-anthropic-last-text (new, test infrastructure)"
  [ResponsesToolChoiceNoneReachesAnthropic]="responses-anthropic-tool-choice-none (new, go2lean M2 agent)"
  [GeminiToolChoiceNoneStaysNone]="gemini-tool-choice-none (new, go2lean M2 agent)"
  [BetaMetadataEmptyAccountNoPanic]="anthropic-metadata-empty-account-panic (new)"
  [BetaBudgetThinkingKeepsToolTurnThinking]="anthropic-budget-thinking-stripped, beta (new)"
  [V1BudgetThinkingKeepsToolTurnThinking]="anthropic-budget-thinking-stripped, v1 (new)"
  [CodexNonStreamKeepsFunctionCall]="codex-nonstream-function-calls-dropped (new)"
  [CountTokensIncludesToolResult]="count-tokens-ignores-tool-blocks (new)"
  [GeminiSchemaExclusiveBoundDeterministic]="gemini-schema-exclusive-bounds, both keys (new)"
  [GeminiSchemaNestedExclusiveBoundRemoved]="gemini-schema-exclusive-bounds, nested (new)"
  [GeminiThinkingKeepsClientExtraBody]="gemini-thinking-extra-body-overwritten (new, latent)"
  [RecordingAssemblerKeepsInputUsage]="recording-anthropic-stream-usage (new, minor)"
  [ResponsesAssemblerToolCallArguments]="responses-assembler-tool-call-arguments (new, latent)"
  [StreamCounterResetClearsUpstream]="stream-counter-reset-upstream (new, latent)"
)
status=0
printf '\n%-40s  %-11s  %s\n' test result finding
for t in "${!finding[@]}"; do
  if grep -q -- "--- FAIL: TestGo2LeanRepro_$t " <<<"$out"; then r=reproduces
  elif grep -q -- "--- PASS: TestGo2LeanRepro_$t " <<<"$out"; then r=fixed
  else r=not-run; status=1
  fi
  printf '%-40s  %-11s  %s\n' "$t" "$r" "${finding[$t]}"
done | sort -k3
if grep -q -E '^(# |FAIL\s.*\[build failed\])' <<<"$out"; then
  echo; echo "build output:"; grep -E -A3 '^# |build failed' <<<"$out" | head -30
  status=1
fi
echo
grep -E '^\s+zz_go2lean_repro\w*_test.go:[0-9]+:' <<<"$out" || true
exit $status
