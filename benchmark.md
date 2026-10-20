# Benchmark: can go2lean find the bugs found by hand?

Findings 6-11 (see [findings/](findings/)) were found by reading code and
checking the bug classes of similar gateways
([hunting-similar-tools.md](hunting-similar-tools.md)), not by go2lean. They
are used as a benchmark for the tool: for each, the general property a
reviewer would state is written as a spec (no trigger input in it), go2lean
searches blind, and `go test` confirms what it finds. A miss names a gap in
go2lean, which is then closed (go2lean `docs/design.md` §9c).

| finding | spec (in `specs/`) | go2lean `bac5535` | what it took |
|---|---|---|---|
| [chat-request-params](findings/chat-request-params.md), sampling and stop | `SpecConvertAnthropicBetaToOpenAIRequestKeepsSampling` | found at trial 3, confirmed | — |
| [chat-request-params](findings/chat-request-params.md), tool_choice | `SpecConvertAnthropicBetaToOpenAIRequestKeepsToolChoice` | found at trial 10, confirmed | — |
| [responses-done-only-arguments](findings/responses-done-only-arguments.md) | `SpecResponsesToChatConverterProcessEventArgumentsAnySchedule` | found at trial 13, confirmed | spec: the event schedule is an input (the earlier stream specs replayed one fixed schedule) |
| (found by go2lean) [chat-anthropic-interleaved-blocks](findings/chat-anthropic-interleaved-blocks.md) | `SpecOpenAIToAnthropicConverterProcessChunkOneOpenBlock` | found at trial 24, confirmed | harness: an empty map is not nil |
| [chat-anthropic-reasoning-null](findings/chat-anthropic-reasoning-null.md), stream | `SpecOpenAIToAnthropicConverterProcessChunkOneOpenTextBlock` | found at trial 80, confirmed (a delta with `reasoning_content` after text) | go2lean: map and interface inputs, `json.Unmarshal`, SDK raw JSON (`RawJSON()`) generated from compared strings, documented enums (`// Any of ...`); harness: SDK values decoded from one JSON document, inline union variants; spec: finish reasons cleared, converter without a token counter (tiktoken out of reach) |
| [chat-anthropic-reasoning-null](findings/chat-anthropic-reasoning-null.md), non-stream | `SpecConvertOpenAIChatToAnthropicBetaThinkingIsText` | go2lean `74f24df`: found at trial 160, confirmed | struct JSON encoding and decoding (`JDesc` tables, go2lean `docs/design.md` §9c); pointers to SDK types described by their element |
| [usage-nonstream-cached-input](findings/usage-nonstream-cached-input.md) | — | out of reach | inline in an HTTP handler |
| [env-credentials-upstream](findings/env-credentials-upstream.md) | — | out of reach | the environment is read inside the SDK |
| [recording-client-credentials](findings/recording-client-credentials.md) | — | no property yet | — |

**5 of 5 reachable cases confirmed** (`scripts/benchmark.sh`, 2026-10-20).
Misses on the way were sorted into three kinds: the spec (a fixed event
schedule; chunks skipped for a random finish reason), go2lean (inputs it
could not generate, calls it did not model, harness renderings that
changed the value), and reach (code the function-level search does not
get to).

## Round 2: findings 13-19

Eight more bugs were found by reading code (findings 13-20, pages in
[findings/](findings/)). Each got a general spec before any search (no
trigger values; each was checked to fail on the hand-found trigger), and
`scripts/benchmark.sh` hunted them blind with go2lean `3191ef8`:

| finding | spec | go2lean `3191ef8` |
|---|---|---|
| [responses-stream-trailing-usage](findings/responses-stream-trailing-usage.md) | `SpecChatToResponsesConverterProcessChunkTerminalUsage` | found at trial 5, confirmed |
| [responses-chat-tool-call-index](findings/responses-chat-tool-call-index.md) | `SpecResponsesToChatConverterProcessEventToolIndexesDense` | found at trial 77, confirmed |
| [responses-developer-message-as-user](findings/responses-developer-message-as-user.md) | `SpecConvertResponsesInputToMessagesKeepsInstructionRoles` | found at trial 127, confirmed |
| [anthropic-abnormal-stop-reasons](findings/anthropic-abnormal-stop-reasons.md), Chat | `SpecConvertAnthropicBetaToOpenAIChatCutShortIsNotStop` | found at trial 7, confirmed |
| [anthropic-abnormal-stop-reasons](findings/anthropic-abnormal-stop-reasons.md), Responses | `SpecConvertAnthropicBetaToResponsesWireCutShortIsIncomplete` | found at trial 2, confirmed |
| [chat-tool-call-empty-id](findings/chat-tool-call-empty-id.md), Responses | `SpecConvertChatToResponsesWireCallIDs` | found at trial 1, confirmed |
| [chat-tool-call-empty-id](findings/chat-tool-call-empty-id.md), Anthropic stream | `SpecOpenAIToAnthropicConverterProcessChunkToolUseIDs` | found at trial 4, confirmed |
| [chat-tool-call-empty-id](findings/chat-tool-call-empty-id.md), Anthropic non-stream | `SpecConvertOpenAIChatToAnthropicBetaToolUseIDs` | missed: stuck in `json.Marshal` (see below); go2lean `74f24df`: found at trial 4, confirmed |
| [chat-anthropic-request-coverage](findings/chat-anthropic-request-coverage.md), developer messages | `SpecConvertOpenAIToAnthropicRequestKeepsInstructions` | found at trial 0, confirmed |
| [chat-anthropic-request-coverage](findings/chat-anthropic-request-coverage.md), tools and tool_choice | `SpecConvertOpenAIToAnthropicRequestKeepsTools` | found at trial 1169, confirmed |
| [chat-responses-request-params](findings/chat-responses-request-params.md) | `SpecConvertChatToOpenAIResponsesKeepsLimits` | found at trial 0, confirmed |
| [upstream-errors-not-translated](findings/upstream-errors-not-translated.md) | — | out of reach: HTTP error handling |

**10 of 11 reachable cases found blind and confirmed**; with the round-1
cases, 15 of 17 (the other miss is the non-stream thinking case above).
Both misses get stuck in the same place: the non-stream Chat -> Anthropic
converter marshals a struct whose `interface{}` member holds a slice of
SDK param unions, and the frontend described every pointer variant
(`*BetaTextBlockParam`, ...) as an opaque custom marshaler, because a
pointer has its element's `MarshalJSON` method. Fixed in go2lean `74f24df`
(pointers described by their element, with the other JSON-model fixes of a
review against Go's and the SDKs' sources): the tool-id case is found at
trial 4 and the round-1 thinking case at trial 160, both confirmed.
**17 of 17 reachable cases.**

Run with `scripts/benchmark.sh <go2lean> <tingly-box> <workspace>` (`REUSE=1` hunts again in a workspace compiled before); the cases are `benchmark.tsv`.
