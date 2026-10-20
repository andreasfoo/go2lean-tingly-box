# go2lean on tingly-box

What [go2lean](https://github.com/andreasfoo/go2lean) found in
[tingly-box](https://github.com/tingly-dev/tingly-box), an LLM gateway that
converts requests and responses between the Anthropic, OpenAI Chat and
OpenAI Responses protocols, with and without streaming.

go2lean translates Go into a Lean model, searches the model for inputs
that break a property (written in Go, as a `Spec…` function), and turns
each candidate into a `go test` on the real code: only what the real run
confirms is a finding. This repository holds the tingly-box side: the
properties, the findings with a plain Go reproduction each, and the
record of every run.

## Versions

| | |
|---|---|
| tingly-box | **`b788a3b`** (main of 2026-10-20): every finding, spec, repro test and the model refer to it; findings 1-5 were first found on `5f83f04` |
| go2lean | the model, coverage table and benchmark: `0a2eeab`; each run's own commit is in [benchmark.md](benchmark.md) and the findings pages |
| Lean model | [`model/`](model): the generated Lean modules and per-package reports |

## Findings

**60 bugs**, each confirmed by a failing `go test` on tingly-box `b788a3b` (except two found by reading, marked in the table): 1 critical, 15 high, 22 medium, 22 low (6 of them latent: in code no gateway path calls). 3 were already tracked upstream ([#1921](https://github.com/tingly-dev/tingly-box/issues/1921), [#1923](https://github.com/tingly-dev/tingly-box/issues/1923), [#1924](https://github.com/tingly-dev/tingly-box/issues/1924)); the rest are new.

Most of them lose or change something a client sent or a provider returned while converting between the Anthropic, OpenAI Chat, OpenAI Responses and Gemini protocols: parameters and content parts dropped (sampling, `tool_choice`, structured output, documents, images, tool schemas), tool calls broken (ids, indexes, arguments), thinking and refusals lost, stop reasons and usage reported wrong. Two go further: the gateway's own environment credentials are sent to every upstream (#6), and a malformed `user_id` panics the Anthropic Beta transform (#35; since upstream `9b15a58` the gateway no longer reaches that transform with the triggering request).

**Reported upstream** as one issue, still marked draft (`[DRAFT]` in its title, not handed over yet), [tingly-dev/tingly-box#2148](https://github.com/tingly-dev/tingly-box/issues/2148) (2026-10-20): every finding below except 1-3 (tracked in their own issues) and 35 (fixed upstream on the gateway path by `9b15a58`), re-checked on upstream main `bc460f6`.

Each row links to its page (root cause with file:line, trigger, observed and expected output, repro test); [findings/README.md](findings/README.md) has the full table with status and evidence, severity legend and how each was found.

| # | severity | finding | path |
|---|---|---|---|
| 6 | critical | [gateway environment credentials sent upstream](findings/env-credentials-upstream.md) | upstream clients |
| 1 | high | [tool-call ids not truncated on one side](findings/issue-1924-tool-call-ids.md) (#1924) | Anthropic -> Chat request |
| 2 | high | [system messages dropped](findings/issue-1921-system-messages.md) (#1921) | Anthropic -> Chat, Anthropic v1 -> Responses requests |
| 3 | high | [streamed tool calls all at index 0; max_tokens not "length"](findings/issue-1923-stream-indexes.md) (#1923) | Anthropic -> Chat stream |
| 8 | high | [DeepSeek-style reasoning_content breaks thinking blocks](findings/chat-anthropic-reasoning-null.md) | Chat -> Anthropic stream and non-stream |
| 9 | high | [tool arguments lost when sent only in `.done`](findings/responses-done-only-arguments.md) | Responses -> Chat and Responses -> Anthropic streams |
| 13 | high | [Responses clients get no usage when streaming from a Chat provider](findings/responses-stream-trailing-usage.md) | Chat -> Responses stream, Responses -> Chat request |
| 14 | high | [Chat tool call index taken from the Responses output index](findings/responses-chat-tool-call-index.md) | Responses -> Chat stream |
| 21 | high | [Anthropic tool schemas lose `$defs`, `additionalProperties` and other keywords](findings/anthropic-tool-schema-keywords.md) | Anthropic -> Chat, Anthropic -> Responses request |
| 22 | high | [structured output formats dropped](findings/structured-output-dropped.md) | Chat <-> Responses <-> Anthropic requests, all six directions |
| 23 | high | [an empty thinking block stays open across the answer](findings/responses-anthropic-stream-open-thinking.md) | Responses -> Anthropic stream |
| 25 | high | [PDF, document and file content dropped](findings/document-content-dropped.md) | Anthropic -> Chat, Anthropic -> Responses, Chat -> Anthropic, Chat -> Responses requests |
| 35 | high | [Anthropic Beta metadata transform panics on a user_id without account](findings/anthropic-metadata-empty-account-panic.md) | Anthropic Beta client -> `api.anthropic.com` provider with an API key |
| 36 | high | [budget-thinking models: thinking stripped from the tool turn while thinking stays on](findings/anthropic-budget-thinking-stripped.md) | Anthropic client -> Anthropic / Claude Code provider (vendor transform), V1 and Beta |
| 37 | high | [Codex without streaming drops function calls](findings/codex-nonstream-function-calls-dropped.md) | Chat and Responses clients -> Codex provider, non-stream |
| 44 | high | [Anthropic -> Gemini drops tool_use arguments](findings/anthropic-gemini-tool-use-args.md) | Anthropic -> Gemini (Google-style providers) request |
| 4 | medium | [reasoning_content empty without streaming](findings/reasoning-content-nonstream.md) | Anthropic -> Chat non-stream |
| 5 | medium | [refusal lost when streaming](findings/responses-chat-stream-refusal.md) | Responses -> Chat stream |
| 7 | medium | [cache reads counted twice without streaming](findings/usage-nonstream-cached-input.md) | usage tracking, Chat passthrough |
| 10 | medium | [Anthropic -> Chat requests drop sampling, stop and tool_choice](findings/chat-request-params.md) | Anthropic -> Chat request |
| 15 | medium | [Responses developer messages become user messages](findings/responses-developer-message-as-user.md) | Responses -> Chat request |
| 16 | medium | [`model_context_window_exceeded` and `refusal` reported as normal completions](findings/anthropic-abnormal-stop-reasons.md) | Anthropic -> Chat non-stream, Anthropic -> Responses stream and non-stream |
| 17 | medium | [Chat tool calls without an id stay without one](findings/chat-tool-call-empty-id.md) | Chat -> Anthropic stream and non-stream, Chat -> Responses non-stream |
| 18 | medium | [Chat -> Anthropic requests drop developer messages, parameterless tools, tool_choice modes](findings/chat-anthropic-request-coverage.md) | Chat -> Anthropic request |
| 19 | medium | [Chat -> Responses requests drop max_completion_tokens, zero sampling values, parallel_tool_calls](findings/chat-responses-request-params.md) | Chat -> Responses request |
| 20 | medium | [upstream errors not translated to the client's protocol](findings/upstream-errors-not-translated.md) | error responses, all converted paths |
| 24 | medium | [parameterless tool calls stream arguments `""`](findings/parameterless-tool-stream-arguments.md) | Anthropic -> Chat and Anthropic -> Responses streams |
| 26 | medium | [a refusal becomes `content: null`](findings/responses-anthropic-nonstream-refusal.md) | Responses -> Anthropic non-stream |
| 33 | medium | [tool_choice "none" becomes auto](findings/responses-anthropic-tool-choice-none.md) | Responses -> Anthropic request |
| 38 | medium | [count_tokens estimate ignores tool_use, tool_result, document and image blocks](findings/count-tokens-ignores-tool-blocks.md) | Anthropic count_tokens, providers without a native endpoint |
| 39 | medium | [Gemini tool schemas: exclusive bounds overwrite minimum/maximum at random, nested schemas not filtered](findings/gemini-schema-exclusive-bounds.md) | Chat -> Gemini (Google, Poe) request |
| 45 | medium | [tool_choice "none" becomes auto](findings/anthropic-tool-choice-none-responses-gemini.md) | Anthropic -> Responses, Anthropic -> Gemini requests |
| 46 | medium | [Chat -> Responses requests drop developer messages](findings/chat-responses-developer-dropped.md) | Chat -> Responses request |
| 47 | medium | [Chat -> Gemini requests drop max_completion_tokens, developer and array-form system text, user images](findings/chat-gemini-request-coverage.md) | Chat -> Gemini (Google-style providers) request |
| 48 | medium | [finish_reason "stop" with tool calls becomes end_turn](findings/chat-stop-with-tool-calls-end-turn.md) | Chat -> Anthropic non-stream and stream |
| 49 | medium | [Gemini thoughtsTokenCount not counted as output](findings/gemini-thoughts-tokens-dropped.md) | Gemini -> Anthropic, Gemini -> Chat responses, usage tracking |
| 50 | medium | [base64 images encoded twice](findings/anthropic-gemini-image-double-encoded.md) | Anthropic -> Gemini (Google-style providers) request |
| 11 | low | [Chat -> Anthropic stream opens a block while another is open](findings/chat-anthropic-interleaved-blocks.md) | Chat -> Anthropic stream |
| 12 | low | [recordings keep the request URL's query string](findings/recording-client-credentials.md) | recording |
| 27 | low | [one assistant turn becomes two assistant messages](findings/responses-chat-split-assistant-turn.md) | Responses -> Chat request |
| 28 | low | [reasoning summaries dropped without streaming](findings/responses-anthropic-nonstream-reasoning-summary.md) | Responses -> Anthropic non-stream |
| 29 | low | [images in tool output dropped](findings/responses-anthropic-tool-output-image.md) | Responses -> Anthropic request |
| 30 | low | [tool `strict` flag lost](findings/chat-responses-tool-strict.md) | Chat -> Responses, Responses -> Chat requests |
| 31 | low | [stream input usage depends on which fields message_delta repeats](findings/usage-anthropic-delta-cache-creation.md) | usage accumulation, Anthropic provider streams |
| 32 | low | [`ParseAnthropicResult` keeps only the last text block](findings/sse-parse-anthropic-last-text.md) | test infrastructure (`sse` parsers) |
| 34 | low | [Gemini function calling mode NONE becomes auto](findings/gemini-tool-choice-none.md) (latent) | Gemini -> Chat, Gemini -> Anthropic request (no gateway caller) |
| 40 | low | [Gemini thinking config replaces `extra_body`](findings/gemini-thinking-extra-body-overwritten.md) (latent) | Chat -> Gemini request (client `extra_body` already dropped at parsing) |
| 41 | low | [recordings of Anthropic streams lose input and cache usage](findings/recording-anthropic-stream-usage.md) | recording, Anthropic streams |
| 42 | low | [`ResponsesAssembler.ToolCalls()` arguments always empty](findings/responses-assembler-tool-call-arguments.md) (latent) | Responses stream assembler (no gateway caller) |
| 43 | low | [`StreamTokenCounter.Reset` keeps the upstream usage](findings/stream-counter-reset-upstream.md) (latent) | streaming token counter (no gateway caller) |
| 51 | low | [disable_parallel_tool_use dropped](findings/anthropic-disable-parallel-tool-use.md) | Anthropic -> Chat, Anthropic -> Responses requests |
| 52 | low | [tool `strict` flag lost](findings/anthropic-chat-tool-strict.md) | Anthropic -> Chat, Anthropic -> Responses, Chat -> Anthropic requests |
| 53 | low | [Chat -> Responses requests drop user, safety_identifier, metadata, service_tier, store, top_logprobs](findings/chat-responses-request-flags.md) | Chat -> Responses request |
| 54 | low | [Claude's thinking dropped for Responses clients](findings/anthropic-responses-thinking-dropped.md) | Anthropic -> Responses non-stream (stream by reading) |
| 55 | low | [a Chat refusal dropped without streaming](findings/chat-responses-refusal-dropped.md) | Chat -> Responses non-stream |
| 56 | low | [Gemini -> Anthropic requests drop sampling parameters and stop sequences](findings/gemini-anthropic-sampling-dropped.md) (latent) | Gemini -> Anthropic request (no gateway caller) |
| 57 | low | [max_tokens above 2^31-1 wraps in maxOutputTokens](findings/gemini-max-tokens-int32-wrap.md) | Anthropic -> Gemini, Chat -> Gemini (Google-style providers) request |
| 58 | low | [Responses -> Chat requests drop parallel_tool_calls](findings/responses-chat-parallel-tool-calls.md) | Responses -> Chat request |
| 59 | low | [Gemini SAFETY becomes stop_reason "content_filter"](findings/gemini-anthropic-safety-content-filter.md) | Gemini -> Anthropic v1 non-stream and stream |
| 60 | low | [Anthropic and Chat -> Gemini responses drop thinking, refusals and cached prompt tokens](findings/gemini-client-responses-coverage.md) (latent) | Anthropic -> Gemini, Chat -> Gemini responses (no gateway caller) |

## Coverage: what go2lean translated into Lean

The generated Lean itself is in [`model/lean/`](model/lean) (one module per
Go package, see [model/README.md](model/README.md)).

Every function of the compiled packages, by what the frontend made of it (`scripts/coverage.py model/reports 0a2eeab` prints this from the reports in [`model/reports/`](model/reports)). *complete*: translated, every call modelled. *partial*: translated, some calls or values are havoc (unmodelled; a counterexample that reads one is still checked by `go test`). *rejected*: not translated, with the reason; a call to it is havoc. Functions in `_test.go` files and closures are left out.

Model of go2lean `0a2eeab` (`compile -deps` of the protocol packages; the
other tingly-box packages are compiled as far as those call into them).

| package | functions | complete | partial | rejected | in Lean IR |
|---|---|---|---|---|---|
| `ai` | 62 | 52 | 10 | 0 | 100% |
| `internal/catalog` | 2 | 0 | 2 | 0 | 100% |
| `internal/protocol` | 133 | 103 | 22 | 8 | 94% |
| `internal/protocol/assembler` | 120 | 99 | 21 | 0 | 100% |
| `internal/protocol/ids` | 6 | 6 | 0 | 0 | 100% |
| `internal/protocol/metaid` | 7 | 4 | 3 | 0 | 100% |
| `internal/protocol/nonstream` | 40 | 28 | 12 | 0 | 100% |
| `internal/protocol/ops` | 91 | 51 | 40 | 0 | 100% |
| `internal/protocol/request` | 175 | 145 | 30 | 0 | 100% |
| `internal/protocol/sse` | 14 | 2 | 12 | 0 | 100% |
| `internal/protocol/stream` | 184 | 116 | 67 | 1 | 99% |
| `internal/protocol/thinking` | 6 | 4 | 2 | 0 | 100% |
| `internal/protocol/token` | 11 | 5 | 6 | 0 | 100% |
| `internal/protocol/usage` | 19 | 18 | 1 | 0 | 100% |
| `internal/protocol/wire` | 21 | 20 | 1 | 0 | 100% |
| `internal/typ` | 1 | 1 | 0 | 0 | 100% |
| **all** | **892** | **654** | **229** | **9** | **99%** |

<details><summary><code>ai</code>: 62 functions</summary>

| function | status | note |
|---|---|---|
| `CredentialBundle.Field` | partial | 1 unmodelled site: lookup in map t2 the model knows nothing about is havoc |
| `NormalizeCredential` | partial | 3 unmodelled sites: len of map[string]string assumed below 16 (not modelled) |
| `OAuthDetail.GetExtraFieldString` | partial | 3 unmodelled sites: lookup in map t2 the model knows nothing about is havoc |
| `OAuthDetail.IsExpired` | partial | 4 unmodelled sites: call time.Parse: t3 is havoc (unmodeled-call) |
| `TokenUsage.ToAnthropicUsageMap` | partial | 6 unmodelled sites: map update t0["input_tokens":string] on a map the model knows nothing about ignored |
| `TokenUsage.ToOpenAIChatUsageMap` | partial | 7 unmodelled sites: map update t1["prompt_tokens":string] on a map the model knows nothing about ignored |
| `TokenUsage.ToOpenAIResponsesUsageMap` | partial | 7 unmodelled sites: map update t1["input_tokens":string] on a map the model knows nothing about ignored |
| `TokenUsage.promptCacheDetails` | partial | 2 unmodelled sites: map update t3["cached_tokens":string] on a map the model knows nothing about ignored |
| `ValidateCredential` | partial | 3 unmodelled sites: call encoding/json.Valid: t57 is havoc (unmodeled-call) |
| `WorkBuddySite.ClientHeaders` | partial | 10 unmodelled sites: map update t1["User-Agent":string] on a map the model knows nothing about ignored |
| `AllowedAPIStyles` | complete |  |
| `AuthType.IsMultiFieldCredential` | complete |  |
| `AuthType.IsValid` | complete |  |
| `CredentialSchema` | complete |  |
| `IsSecretCredentialField` | complete |  |
| `IsWorkBuddyIssuer` | complete |  |
| `IssuerSupportsDual` | complete |  |
| `NewTokenUsage` | complete |  |
| `NewTokenUsageFull` | complete |  |
| `NewTokenUsageWithCache` | complete |  |
| `NewTokenUsageWithCacheDetails` | complete |  |
| `OAuthDetail.GetIssuer` | complete |  |
| `OAuthDetail.MarshalJSON` | complete |  |
| `OAuthDetail.UnmarshalJSON` | complete |  |
| `OpenAIEndpointModeForIssuer` | complete |  |
| `Provider.GetAccessToken` | complete |  |
| `Provider.HasDualURL` | complete |  |
| `Provider.IsAPIKey` | complete |  |
| `Provider.IsBuiltin` | complete |  |
| `Provider.IsClaudeCodeProvider` | complete |  |
| `Provider.IsCodexProvider` | complete |  |
| `Provider.IsDual` | complete |  |
| `Provider.IsMultiFieldCredential` | complete |  |
| `Provider.IsOAuth` | complete |  |
| `Provider.IsOAuthExpired` | complete |  |
| `Provider.IsOAuthToken` | complete |  |
| `Provider.IsVirtual` | complete |  |
| `Provider.OAuthIssuer` | complete |  |
| `Provider.ResolveEndpoint` | complete |  |
| `Provider.ResolveStyle` | complete |  |
| `Provider.dualEligible` | complete |  |
| `TokenUsage.HasCacheUsage` | complete |  |
| `TokenUsage.HasCacheUsage~NewTokenUsage` | complete |  |
| `TokenUsage.HasUsage` | complete |  |
| `TokenUsage.HasUsage~NewTokenUsage` | complete |  |
| `TokenUsage.PromptTotalTokens` | complete |  |
| `TokenUsage.PromptTotalTokens~NewTokenUsage` | complete |  |
| `TokenUsage.ToAnthropicMessageDeltaUsageMap` | complete |  |
| `TokenUsage.ToAnthropicMessageDeltaUsageMap~NewTokenUsage` | complete |  |
| `TokenUsage.ToAnthropicUsageMap~NewTokenUsage` | complete |  |
| `TokenUsage.ToOpenAIChatUsageMap~NewTokenUsage` | complete |  |
| `TokenUsage.ToOpenAIResponsesUsageMap~NewTokenUsage` | complete |  |
| `TokenUsage.TotalTokens` | complete |  |
| `TokenUsage.TotalTokens~NewTokenUsage` | complete |  |
| `TokenUsage.UncachedInputTokens` | complete |  |
| `TokenUsage.UncachedInputTokens~NewTokenUsage` | complete |  |
| `TokenUsage.promptCacheDetails~NewTokenUsage` | complete |  |
| `ValidateCredentialAPIStyle` | complete |  |
| `WorkBuddySite.APIBase` | complete |  |
| `WorkBuddySiteFor` | complete |  |
| `ZCodeEndpoints` | complete |  |
| `ZeroTokenUsage` | complete |  |

</details>

<details><summary><code>internal/catalog</code>: 2 functions</summary>

| function | status | note |
|---|---|---|
| `ClaudeThinkingCaps.SupportsEffort` | partial | 1 unmodelled site: len of map[string]bool assumed below 16 (not modelled) |
| `LookupClaudeThinkingCaps` | partial | 1 unmodelled site: load claudeCapsIndex: t2 is havoc (unmodeled-load) |

</details>

<details><summary><code>internal/protocol</code>: 133 functions</summary>

| function | status | note |
|---|---|---|
| `HandleContext.ProcessStream` | rejected | block 0 (entry): defer (*HandleContext).ReleaseStreamState(t4): deferred call to ReleaseStreamState (defer of a modelled function) |
| `ResponseInputItemUnionParam.GetExecution` | rejected | block 3 (if.then): changetype *string <- *ResponseToolSearchOutputItemParamExecution (t8): conversion *github.com/openai/openai-go/v3/res... |
| `ResponseInputItemUnionParam.GetPhase` | rejected | block 1 (if.then): changetype *string <- *EasyInputMessagePhase (t4): conversion *github.com/openai/openai-go/v3/responses.EasyInputMessa... |
| `ResponseInputItemUnionParam.GetRole` | rejected | block 1 (if.then): changetype *string <- *EasyInputMessageRole (t4): conversion *github.com/openai/openai-go/v3/responses.EasyInputMessag... |
| `ResponseInputItemUnionParam.GetStatus` | rejected | block 3 (if.then): changetype *string <- *ResponseOutputMessageStatus (t8): conversion *github.com/openai/openai-go/v3/responses.Response... |
| `ResponseInputItemUnionParam.GetType` | rejected | block 1 (if.then): changetype *string <- *EasyInputMessageType (t4): conversion *github.com/openai/openai-go/v3/responses.EasyInputMessag... |
| `ResponseNewParams.MarshalJSON` | rejected | block 0 (entry): changetype *shadow <- *ResponseNewParams (t0): conversion *github.com/openai/openai-go/v3/responses.ResponseNewParams ->... |
| `RunLoop` | rejected | type assertion t2 on an interface value the model did not build is havoc |
| `AnthropicBetaMessagesRequest.MarshalJSON` | partial | 1 unmodelled site: map update t3["stream":string] on a map the model knows nothing about ignored |
| `AnthropicMessagesRequest.MarshalJSON` | partial | 1 unmodelled site: map update t3["stream":string] on a map the model knows nothing about ignored |
| `ChatModerationFromResponses` | partial | 2 unmodelled sites: map update t5["input":string] on a map the model knows nothing about ignored |
| `ChatServiceTierFromAnthropic` | partial | 1 unmodelled site: conversion T -> string: t1 is havoc (opaque-op) |
| `ClassifyTransportError` | partial | 7 unmodelled sites: load Canceled: t1 is havoc (unmodeled-load) |
| `ClassifyUpstreamFailure` | partial | 5 unmodelled sites: call errors.As: t6 is havoc (unmodeled-call) |
| `CommitFirstChunk` | partial | 2 unmodelled sites: type assertion t2 on an interface value the model did not build is havoc |
| `IsContextCanceled` | partial | 2 unmodelled sites: load Canceled: t0 is havoc (unmodeled-load) |
| `MarkFirstToken` | partial | 1 unmodelled site: call time.Now: t5 is havoc (unmodeled-call) |
| `OpenAIChatCompletionRequest.MarshalJSON` | partial | 1 unmodelled site: map update t13["stream":string] on a map the model knows nothing about ignored |
| `PreprocessInputData` | partial | 4 unmodelled sites: len of map[int]string assumed below 16 (not modelled) |
| `SnapshotJSON` | partial | 7 unmodelled sites: type assertion t5 on an interface value the model did not build is havoc |
| `TokenUsage.ToAnthropicUsageMap` | partial | 6 unmodelled sites: map update t0["input_tokens":string] on a map the model knows nothing about ignored |
| `TokenUsage.ToOpenAIChatUsageMap` | partial | 7 unmodelled sites: map update t1["prompt_tokens":string] on a map the model knows nothing about ignored |
| `TokenUsage.ToOpenAIResponsesUsageMap` | partial | 7 unmodelled sites: map update t1["input_tokens":string] on a map the model knows nothing about ignored |
| `TokenUsage.promptCacheDetails` | partial | 2 unmodelled sites: map update t3["cached_tokens":string] on a map the model knows nothing about ignored |
| `UpstreamLogDetail` | partial | 2 unmodelled sites: call errors.As: t2 is havoc (unmodeled-call) |
| `chatModerationSide` | partial | 3 unmodelled sites: map update t4["type":string] on a map the model knows nothing about ignored |
| `isDNSError` | partial | 1 unmodelled site: call errors.As: t2 is havoc (unmodeled-call) |
| `isNilJSONValue` | partial | 3 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `statusLine` | partial | 1 unmodelled site: call net/http.StatusText: t0 is havoc (unmodeled-call) |
| `truncateClientDetail` | partial | 1 unmodelled site: call unicode/utf8.RuneStart: t9 is havoc (unmodeled-call) |
| `AnthropicBetaMessagesRequest.UnmarshalJSON` | complete |  |
| `AnthropicMessagesRequest.UnmarshalJSON` | complete |  |
| `ChatServiceTierFromAnthropic[github.com/anthropics/anthropic-sdk-go.BetaUsageServiceTier]` | complete |  |
| `ChatServiceTierFromResponses` | complete |  |
| `EmptyResponseError.Error` | complete |  |
| `EmptyResponseError.Unwrap` | complete |  |
| `GetInputValue` | complete |  |
| `Grouper.GroupBeta` | complete |  |
| `Grouper.GroupBeta~NewGrouper` | complete |  |
| `Grouper.GroupV1` | complete |  |
| `Grouper.GroupV1~NewGrouper` | complete |  |
| `Grouper.IsPureBetaUserMessage` | complete |  |
| `Grouper.IsPureBetaUserMessage~NewGrouper` | complete |  |
| `Grouper.IsPureUserMessage` | complete |  |
| `Grouper.IsPureUserMessage~NewGrouper` | complete |  |
| `Grouper.analyzeBetaRound` | complete |  |
| `Grouper.analyzeBetaRound~NewGrouper` | complete |  |
| `Grouper.analyzeV1Round` | complete |  |
| `Grouper.analyzeV1Round~NewGrouper` | complete |  |
| `HandleContext.CallOnStreamComplete` | complete |  |
| `HandleContext.CallOnStreamComplete~NewHandleContext` | complete |  |
| `HandleContext.DispatchStreamError` | complete |  |
| `HandleContext.DispatchStreamError~NewHandleContext` | complete |  |
| `HandleContext.ReleaseStreamState` | complete |  |
| `HandleContext.ReleaseStreamState~NewHandleContext` | complete |  |
| `HandleContext.SendError` | complete |  |
| `HandleContext.SendError~NewHandleContext` | complete |  |
| `HandleContext.SetupSSEHeaders` | complete |  |
| `HandleContext.SetupSSEHeaders~NewHandleContext` | complete |  |
| `HandleContext.WithOnStreamComplete` | complete |  |
| `HandleContext.WithOnStreamComplete~NewHandleContext` | complete |  |
| `HandleContext.WithOnStreamError` | complete |  |
| `HandleContext.WithOnStreamError~NewHandleContext` | complete |  |
| `HandleContext.WithOnStreamEvent` | complete |  |
| `HandleContext.WithOnStreamEvent~NewHandleContext` | complete |  |
| `LoggableUpstreamError` | complete |  |
| `NewGrouper` | complete |  |
| `NewHandleContext` | complete |  |
| `OpenAIChatCompletionRequest.UnmarshalJSON` | complete |  |
| `Response.OutputText` | complete |  |
| `Response.RawJSON` | complete |  |
| `Response.UnmarshalJSON` | complete |  |
| `ResponseCreateRequest.UnmarshalJSON` | complete |  |
| `ResponseInputItemUnionParam.GetAcknowledgedSafetyChecks` | complete |  |
| `ResponseInputItemUnionParam.GetAction` | complete |  |
| `ResponseInputItemUnionParam.GetActions` | complete |  |
| `ResponseInputItemUnionParam.GetApprovalRequestID` | complete |  |
| `ResponseInputItemUnionParam.GetApprove` | complete |  |
| `ResponseInputItemUnionParam.GetArguments` | complete |  |
| `ResponseInputItemUnionParam.GetAsync` | complete |  |
| `ResponseInputItemUnionParam.GetBackground` | complete |  |
| `ResponseInputItemUnionParam.GetCallID` | complete |  |
| `ResponseInputItemUnionParam.GetCaller` | complete |  |
| `ResponseInputItemUnionParam.GetCode` | complete |  |
| `ResponseInputItemUnionParam.GetContainerID` | complete |  |
| `ResponseInputItemUnionParam.GetContent` | complete |  |
| `ResponseInputItemUnionParam.GetEncryptedContent` | complete |  |
| `ResponseInputItemUnionParam.GetEnvironment` | complete |  |
| `ResponseInputItemUnionParam.GetError` | complete |  |
| `ResponseInputItemUnionParam.GetFingerprint` | complete |  |
| `ResponseInputItemUnionParam.GetID` | complete |  |
| `ResponseInputItemUnionParam.GetInput` | complete |  |
| `ResponseInputItemUnionParam.GetMaxOutputLength` | complete |  |
| `ResponseInputItemUnionParam.GetName` | complete |  |
| `ResponseInputItemUnionParam.GetNamespace` | complete |  |
| `ResponseInputItemUnionParam.GetOperation` | complete |  |
| `ResponseInputItemUnionParam.GetOutput` | complete |  |
| `ResponseInputItemUnionParam.GetOutputFormat` | complete |  |
| `ResponseInputItemUnionParam.GetOutputs` | complete |  |
| `ResponseInputItemUnionParam.GetPendingSafetyChecks` | complete |  |
| `ResponseInputItemUnionParam.GetQuality` | complete |  |
| `ResponseInputItemUnionParam.GetQueries` | complete |  |
| `ResponseInputItemUnionParam.GetReason` | complete |  |
| `ResponseInputItemUnionParam.GetReasoning` | complete |  |
| `ResponseInputItemUnionParam.GetResult` | complete |  |
| `ResponseInputItemUnionParam.GetResults` | complete |  |
| `ResponseInputItemUnionParam.GetRevisedPrompt` | complete |  |
| `ResponseInputItemUnionParam.GetServerLabel` | complete |  |
| `ResponseInputItemUnionParam.GetSize` | complete |  |
| `ResponseInputItemUnionParam.GetSummary` | complete |  |
| `ResponseInputItemUnionParam.GetTools` | complete |  |
| `ResponseInputItemUnionParam.MarshalJSON` | complete |  |
| `ResponseInputItemUnionParam.UnmarshalJSON` | complete |  |
| `ResponseNewParams.UnmarshalJSON` | complete |  |
| `ResponseNewParamsInputUnion.MarshalJSON` | complete |  |
| `ResponseNewParamsInputUnion.UnmarshalJSON` | complete |  |
| `TokenUsage.HasCacheUsage` | complete |  |
| `TokenUsage.HasUsage` | complete |  |
| `TokenUsage.PromptTotalTokens` | complete |  |
| `TokenUsage.ToAnthropicMessageDeltaUsageMap` | complete |  |
| `TokenUsage.TotalTokens` | complete |  |
| `TokenUsage.UncachedInputTokens` | complete |  |
| `UpstreamMessage` | complete |  |
| `UpstreamStatus` | complete |  |
| `commitFirstChunk` | complete |  |
| `flattenOutputTextContent` | complete |  |
| `isTLSError` | complete |  |
| `loggableUpstreamError.Error` | complete |  |
| `loggableUpstreamError.Unwrap` | complete |  |
| `safeAnthropicMessage` | complete |  |
| `safeOpenAIMessage` | complete |  |
| `truncateLogDetail` | complete |  |
| `withSDKStatus` | complete |  |

</details>

<details><summary><code>internal/protocol/assembler</code>: 120 functions</summary>

| function | status | note |
|---|---|---|
| `AnthropicStreamAssembler.Finish` | partial | 9 unmodelled sites: len of map[int]github.com/anthropics/anthropic-sdk-go.ContentBlockUnion assumed below 16 (not modelled) |
| `AnthropicStreamAssembler.handleContentBlockDelta` | partial | 12 unmodelled sites: lookup in map t2 the model knows nothing about is havoc |
| `AnthropicStreamAssembler.handleContentBlockDeltaBeta` | partial | 12 unmodelled sites: lookup in map t2 the model knows nothing about is havoc |
| `AnthropicStreamAssembler.handleContentBlockStart` | partial | 2 unmodelled sites: map update t20[blockIndex] on a map the model knows nothing about ignored |
| `AnthropicStreamAssembler.handleContentBlockStartBeta` | partial | 2 unmodelled sites: map update t20[blockIndex] on a map the model knows nothing about ignored |
| `NewResponsesAssembler` | partial | 2 unmodelled sites: call time.Now: t7 is havoc (unmodeled-call) |
| `NewResponsesAssemblerWithID` | partial | 2 unmodelled sites: call time.Now: t9 is havoc (unmodeled-call) |
| `OpenAIChatChoiceCount` | partial | 2 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `ResponsesAssembler.Accumulate` | partial | 30 unmodelled sites: lookup in map t22 the model knows nothing about is havoc |
| `ResponsesAssembler.HasImage` | partial | 1 unmodelled site: len of map[int]*github.com/tingly-dev/tingly-box/internal/protocol/assembler.pendingImageData assumed below 16 (not mo... |
| `ResponsesAssembler.HasImageAt` | partial | 4 unmodelled sites: lookup in map t2 the model knows nothing about is havoc |
| `ResponsesAssembler.ImageCallIDAt` | partial | 4 unmodelled sites: lookup in map t2 the model knows nothing about is havoc |
| `ResponsesAssembler.ImageCallIDs` | partial | 3 unmodelled sites: len of map[int]*github.com/tingly-dev/tingly-box/internal/protocol/assembler.pendingImageData assumed below 16 (not m... |
| `ResponsesAssembler.ImageCount` | partial | 1 unmodelled site: len of map[int]*github.com/tingly-dev/tingly-box/internal/protocol/assembler.pendingImageData assumed below 16 (not mo... |
| `ResponsesAssembler.ImageDataAt` | partial | 4 unmodelled sites: lookup in map t2 the model knows nothing about is havoc |
| `ResponsesAssembler.Images` | partial | 3 unmodelled sites: len of map[int]*github.com/tingly-dev/tingly-box/internal/protocol/assembler.pendingImageData assumed below 16 (not m... |
| `ResponsesAssembler.ToolCalls` | partial | 2 unmodelled sites: range over map t3: in insertion order; a map the model knows nothing about has no entries |
| `ResponsesAssembler.syntheticResponse` | partial | 2 unmodelled sites: syntheticResponse: conversion int64 -> float64: assumed the zero value of float64 (not modelled) |
| `openAIChatStreamAssembler.Add` | partial | 2 unmodelled sites: nil comparison t38 on a value of unknown nil-ness is havoc |
| `openAIChatStreamAssembler.Terminal` | partial | 1 unmodelled site: len of map[int64]struct{} assumed below 16 (not modelled) |
| `streamEventJSON` | partial | 1 unmodelled site: nil comparison t0 on a value of unknown nil-ness is havoc |
| `AnthropicBetaSDKAssembler.Accumulate` | complete |  |
| `AnthropicBetaSDKAssembler.Accumulate~NewAnthropicBetaSDKAssembler` | complete |  |
| `AnthropicBetaSDKAssembler.Finish` | complete |  |
| `AnthropicBetaSDKAssembler.Finish~NewAnthropicBetaSDKAssembler` | complete |  |
| `AnthropicBetaSDKAssembler.Result` | complete |  |
| `AnthropicBetaSDKAssembler.Result~NewAnthropicBetaSDKAssembler` | complete |  |
| `AnthropicSDKAssembler.Accumulate` | complete |  |
| `AnthropicSDKAssembler.Accumulate~NewAnthropicSDKAssembler` | complete |  |
| `AnthropicSDKAssembler.Finish` | complete |  |
| `AnthropicSDKAssembler.Finish~NewAnthropicSDKAssembler` | complete |  |
| `AnthropicSDKAssembler.Result` | complete |  |
| `AnthropicSDKAssembler.Result~NewAnthropicSDKAssembler` | complete |  |
| `AnthropicStreamAssembler.Finish~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.RecordV1BetaEvent` | complete |  |
| `AnthropicStreamAssembler.RecordV1BetaEvent~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.RecordV1Event` | complete |  |
| `AnthropicStreamAssembler.RecordV1Event~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.SetUsage` | complete |  |
| `AnthropicStreamAssembler.SetUsageFromTokenUsage` | complete |  |
| `AnthropicStreamAssembler.SetUsageFromTokenUsage~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.SetUsage~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.handleContentBlockDeltaBeta~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.handleContentBlockDelta~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.handleContentBlockStartBeta~NewAnthropicStreamAssembler` | complete |  |
| `AnthropicStreamAssembler.handleContentBlockStart~NewAnthropicStreamAssembler` | complete |  |
| `NewAnthropicBetaSDKAssembler` | complete |  |
| `NewAnthropicSDKAssembler` | complete |  |
| `NewAnthropicStreamAssembler` | complete |  |
| `NewOpenAIStreamAssembler` | complete |  |
| `NewStreamAssembler` | complete |  |
| `NewStreamAssemblerForRequest` | complete |  |
| `OpenAIChatStreamAssembler.AddChunk` | complete |  |
| `OpenAIChatStreamAssembler.Finish` | complete |  |
| `OpenAIChatStreamAssembler.JustFinishedContent` | complete |  |
| `OpenAIChatStreamAssembler.JustFinishedRefusal` | complete |  |
| `OpenAIChatStreamAssembler.JustFinishedToolCall` | complete |  |
| `OpenAIChatStreamAssembler.Result` | complete |  |
| `ResponsesAssembler.Accumulate~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.CurrentRefusal` | complete |  |
| `ResponsesAssembler.CurrentRefusal~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.CurrentText` | complete |  |
| `ResponsesAssembler.CurrentText~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.Finish` | complete |  |
| `ResponsesAssembler.Finish~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.GetOrCreateItemID` | complete |  |
| `ResponsesAssembler.GetOrCreateItemID~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.GetOrCreateResponseID` | complete |  |
| `ResponsesAssembler.GetOrCreateResponseID~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.HasImageAt~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.HasImage~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.ImageCallIDAt~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.ImageCallIDs~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.ImageCount~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.ImageDataAt~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.Images~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.IsCompleted` | complete |  |
| `ResponsesAssembler.IsCompleted~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.IsFailed` | complete |  |
| `ResponsesAssembler.IsFailed~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.IsFinished` | complete |  |
| `ResponsesAssembler.IsFinished~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.IsIncomplete` | complete |  |
| `ResponsesAssembler.IsIncomplete~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.OutputText` | complete |  |
| `ResponsesAssembler.OutputText~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.Response` | complete |  |
| `ResponsesAssembler.ResponseID` | complete |  |
| `ResponsesAssembler.ResponseID~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.Response~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.SetItemID` | complete |  |
| `ResponsesAssembler.SetItemID~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.SetResponseID` | complete |  |
| `ResponsesAssembler.SetResponseID~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.Status` | complete |  |
| `ResponsesAssembler.Status~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.ToolCalls~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.ensureResponseHasAccumulatedOutput` | complete |  |
| `ResponsesAssembler.ensureResponseHasAccumulatedOutput~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.nextSequenceNumber` | complete |  |
| `ResponsesAssembler.nextSequenceNumber~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.syntheticOutputItems` | complete |  |
| `ResponsesAssembler.syntheticOutputItems~NewResponsesAssembler` | complete |  |
| `ResponsesAssembler.syntheticResponse~NewResponsesAssembler` | complete |  |
| `anthropicBetaStreamAssembler.Add` | complete |  |
| `anthropicBetaStreamAssembler.Finish` | complete |  |
| `anthropicBetaStreamAssembler.Terminal` | complete |  |
| `anthropicBetaStreamAssembler.TerminalError` | complete |  |
| `anthropicV1StreamAssembler.Add` | complete |  |
| `anthropicV1StreamAssembler.Finish` | complete |  |
| `anthropicV1StreamAssembler.Terminal` | complete |  |
| `anthropicV1StreamAssembler.TerminalError` | complete |  |
| `newStreamAssembler` | complete |  |
| `openAIChatStreamAssembler.Finish` | complete |  |
| `openAIChatStreamAssembler.TerminalError` | complete |  |
| `openAIResponsesStreamAssembler.Add` | complete |  |
| `openAIResponsesStreamAssembler.Finish` | complete |  |
| `openAIResponsesStreamAssembler.Terminal` | complete |  |
| `openAIResponsesStreamAssembler.TerminalError` | complete |  |
| `responseHasPayload` | complete |  |

</details>

<details><summary><code>internal/protocol/ids</code>: 6 functions</summary>

| function | status | note |
|---|---|---|
| `Call` | complete |  |
| `FunctionCall` | complete |  |
| `Message` | complete |  |
| `New` | complete |  |
| `Response` | complete |  |
| `randomHex` | complete |  |

</details>

<details><summary><code>internal/protocol/metaid</code>: 7 functions</summary>

| function | status | note |
|---|---|---|
| `BuildMetadataUserID` | partial | 1 unmodelled site: deferred call github.com/tingly-dev/tingly-box/internal/protocol/metaid.BuildMetadataUserID$1() ignored (effects not m... |
| `MetadataUserID.Fix` | partial | 6 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `ParseMetadataUserID` | partial | 4 unmodelled sites: ParseMetadataUserID: load legacyUserIDRegex: assumed a pointer to a fresh regexp.Regexp with havoc fields (not modelled) |
| `FormatMetadataUserID` | complete |  |
| `MetadataUserID.Format` | complete |  |
| `MetadataUserID.IsEmpty` | complete |  |
| `MetadataUserID.IsValid` | complete |  |

</details>

<details><summary><code>internal/protocol/nonstream</code>: 40 functions</summary>

| function | status | note |
|---|---|---|
| `ConvertAnthropicBetaToOpenAIChat` | partial | 2 unmodelled sites: call time.Now: t74 is havoc (unmodeled-call) |
| `ConvertAnthropicBetaToResponsesWire` | partial | 2 unmodelled sites: call time.Now: t77 is havoc (unmodeled-call) |
| `ConvertChatToResponsesWire` | partial | 2 unmodelled sites: call time.Now: t70 is havoc (unmodeled-call) |
| `ConvertOpenAIChatToAnthropicBeta` | partial | 2 unmodelled sites: nil comparison t67 on a value of unknown nil-ness is havoc |
| `ConvertResponsesToOpenAIChat` | partial | 1 unmodelled site: conversion float64 -> int64: t31 is havoc (opaque-op) |
| `HandleGoogleToAnthropic` | partial | 22 unmodelled sites: map update t2["id":string] on a map the model knows nothing about ignored |
| `HandleGoogleToAnthropicBeta` | partial | 22 unmodelled sites: map update t1["id":string] on a map the model knows nothing about ignored |
| `HandleGoogleToOpenAI` | partial | 30 unmodelled sites: map update t15["role":string] on a map the model knows nothing about ignored |
| `WriteAnthropicMessage` | partial | 3 unmodelled sites: type assertion: t0 is havoc (opaque-assert) |
| `anthropicMessageModel` | partial | 4 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `marshalOpenAIChatToAnthropic` | partial | 2 unmodelled sites: nil comparison t67 on a value of unknown nil-ness is havoc |
| `resolveResponsesArguments` | partial | 1 unmodelled site: nil comparison t11 on a value of unknown nil-ness is havoc |
| `BuildOpenAIChatPayloadFromResponses` | complete |  |
| `BuildResponsesPayloadFromAnthropicBeta` | complete |  |
| `BuildResponsesPayloadFromChat` | complete |  |
| `ConvertOpenAIChatToAnthropicV1` | complete |  |
| `HandleAnthropicBetaToOpenAIResponse` | complete |  |
| `HandleAnthropicBetaToResponses` | complete |  |
| `HandleAnthropicToGoogle` | complete |  |
| `HandleAnthropicV1` | complete |  |
| `HandleAnthropicV1Beta` | complete |  |
| `HandleOpenAIChat` | complete |  |
| `HandleOpenAIChatToAnthropic` | complete |  |
| `HandleOpenAIChatToResponses` | complete |  |
| `HandleOpenAIResponses` | complete |  |
| `HandleOpenAIToGoogle` | complete |  |
| `HandleResponsesToAnthropicBeta` | complete |  |
| `HandleResponsesToOpenAIChat` | complete |  |
| `MapAnthropicFinishReasonToGoogle` | complete |  |
| `MapGoogleFinishReasonToAnthropic` | complete |  |
| `MapGoogleFinishReasonToAnthropicBeta` | complete |  |
| `MapGoogleFinishReasonToOpenAI` | complete |  |
| `MapOpenAIFinishReasonToGoogle` | complete |  |
| `anthropicStopReasonToResponsesStatus` | complete |  |
| `anthropicUsageWire` | complete |  |
| `buildResponsesToChatNonStreamState` | complete |  |
| `chatFinishReasonToResponsesStatus` | complete |  |
| `firstNonEmpty` | complete |  |
| `mapResponsesFinishReason` | complete |  |
| `responsesToolCallID` | complete |  |

</details>

<details><summary><code>internal/protocol/ops</code>: 91 functions</summary>

| function | status | note |
|---|---|---|
| `ApplyAnthropicBetaModelTransform` | partial | 1 unmodelled site: lookup in map t15 the model knows nothing about is havoc |
| `ApplyAnthropicV1ModelTransform` | partial | 1 unmodelled site: lookup in map t15 the model knows nothing about is havoc |
| `ApplyCodexResponsesTransform` | partial | 2 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `ApplyCursorCompatContentNormalization` | partial | 4 unmodelled sites: lookup in map t11 the model knows nothing about is havoc |
| `ApplyThinkingEffort` | partial | 1 unmodelled site: lookup in map t3 the model knows nothing about is havoc |
| `BuildClaudeCodeBillingHeader` | partial | 6 unmodelled sites: lookup in map t11 the model knows nothing about is havoc |
| `ClaudeCodeVersionFromExtra` | partial | 3 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `FitAnthropicThinkingBudget` | partial | 1 unmodelled site: builtin min: t5 is havoc (unmodeled-call) |
| `SplitProviderHostPath` | partial | 3 unmodelled sites: SplitProviderHostPath: call net/url.Parse #0: assumed a pointer to a fresh net/url.URL with havoc fields (not modelled) |
| `applyCodexToolTransforms` | partial | 1 unmodelled site: lookup in map t14 the model knows nothing about is havoc |
| `applyDeepSeekResponseTransform` | partial | 7 unmodelled sites: lookup in map resp the model knows nothing about is havoc |
| `applyDeepSeekTransform` | partial | 3 unmodelled sites: load deepSeekEffortTiers: t2 is havoc (unmodeled-load) |
| `applyDefaultTransform` | partial | 5 unmodelled sites: nil comparison t16 on a value of unknown nil-ness is havoc |
| `applyGeminiThinkingConfig` | partial | 10 unmodelled sites: lookup in map t3 the model knows nothing about is havoc |
| `applyGeminiToolSchemaFilter` | partial | 1 unmodelled site: len of github.com/openai/openai-go/v3/shared.FunctionParameters assumed below 16 (not modelled) |
| `applyKimiTransform` | partial | 1 unmodelled site: load kimiEffortTiers: t0 is havoc (unmodeled-load) |
| `applyReasoningEffortTier` | partial | 1 unmodelled site: lookup in map tiers the model knows nothing about is havoc |
| `buildGeminiThinkingConfig` | partial | 4 unmodelled sites: map update t2["thinking_budget":string] on a map the model knows nothing about ignored |
| `buildNativeMetadataUserID` | partial | 4 unmodelled sites: lookup in map extra the model knows nothing about is havoc |
| `buildShortNameMap` | partial | 2 unmodelled sites: map update t12[t11] on a map the model knows nothing about ignored |
| `clampAnthropicEffort` | partial | 16 unmodelled sites: clampAnthropicEffort: load anthropicEffortLadder: assumed a slice of 0..3 zero values of github.com/anthropics/anthr... |
| `compactParts` | partial | 1 unmodelled site: call to (*strings.Builder).Grow ignored (no result, effects not modelled) |
| `compactParts[github.com/openai/openai-go/v3.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion]` | partial | 1 unmodelled site: call to (*strings.Builder).Grow ignored (no result, effects not modelled) |
| `compactParts[github.com/openai/openai-go/v3.ChatCompletionContentPartTextParam]` | partial | 1 unmodelled site: call to (*strings.Builder).Grow ignored (no result, effects not modelled) |
| `compactParts[github.com/openai/openai-go/v3.ChatCompletionContentPartUnionParam]` | partial | 1 unmodelled site: call to (*strings.Builder).Grow ignored (no result, effects not modelled) |
| `computeFingerprint` | partial | 2 unmodelled sites: computeFingerprint: call crypto/sha256.Sum256: assumed the zero value of [32]byte (not modelled) |
| `computeFingerprintJS` | partial | 5 unmodelled sites: computeFingerprintJS: call unicode/utf16.Encode: assumed a slice of 0..3 zero values of uint16 (not modelled) |
| `convertThinkingToReasoningContent` | partial | 6 unmodelled sites: nil comparison t21 on a value of unknown nil-ness is havoc |
| `deepSeekThinkingEnabled` | partial | 4 unmodelled sites: lookup in map t3 the model knows nothing about is havoc |
| `disableThinking` | partial | 8 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `enableThinking` | partial | 8 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `filterGeminiProperties` | partial | 4 unmodelled sites: range over map props: in insertion order; a map the model knows nothing about has no entries |
| `filterGeminiSchema` | partial | 15 unmodelled sites: range over map schema: in insertion order; a map the model knows nothing about has no entries |
| `flattenRichContent` | partial | 6 unmodelled sites: type assertion t8 on an interface value the model did not build is havoc |
| `genericEffortTiers` | partial | 6 unmodelled sites: map update t0["minimal":github.com/tingly-dev/tingly-box/internal/protocol/thinking.Level] on a map the model knows n... |
| `getThinkingBudget` | partial | 2 unmodelled sites: lookup in map t0 the model knows nothing about is havoc |
| `lowHighMaxEffortTiers` | partial | 6 unmodelled sites: map update t0["minimal":github.com/tingly-dev/tingly-box/internal/protocol/thinking.Level] on a map the model knows n... |
| `parseBillingHeaderFields` | partial | 1 unmodelled site: call strings.Cut: t17 is havoc (unmodeled-call) |
| `resolveGeminiEffort` | partial | 11 unmodelled sites: lookup in map blob the model knows nothing about is havoc |
| `stripOpenAIThinkingExtra` | partial | 2 unmodelled sites: nil comparison t4 on a value of unknown nil-ness is havoc |
| `ApplyAnthropicBetaDeepSeekThinkingPatch` | complete |  |
| `ApplyAnthropicBetaMetadataTransform` | complete |  |
| `ApplyAnthropicV1DeepSeekThinkingPatch` | complete |  |
| `ApplyAnthropicV1MetadataTransform` | complete |  |
| `ApplyClaudeCodeBetaCompatRoleRewrite` | complete |  |
| `ApplyClaudeCodeCompatRoleRewrite` | complete |  |
| `ApplyProviderTransforms` | complete |  |
| `ApplyResponseTransforms` | complete |  |
| `GenHex5` | complete |  |
| `IsBillingHeaderText` | complete |  |
| `SanitizeAnthropicBetaThinkingConfig` | complete |  |
| `SanitizeAnthropicV1ThinkingConfig` | complete |  |
| `acceptsChatArrayTextContent` | complete |  |
| `anthropicModelThinkingCaps` | complete |  |
| `anthropicOutputEffort` | complete |  |
| `applyCodexThinkingConfig` | complete |  |
| `applyGeminiOpenRouterTransform` | complete |  |
| `applyGeminiPoeTransform` | complete |  |
| `applyGeminiSubsetTransform` | complete |  |
| `applyGeminiTransform` | complete |  |
| `applyNativeClaudeCodeIdentityBeta` | complete |  |
| `applyNativeClaudeCodeIdentityV1` | complete |  |
| `assistantPartText` | complete |  |
| `compactOpenAIChatTextContent` | complete |  |
| `computeCCVersion` | complete |  |
| `computeCCVersionFor` | complete |  |
| `concatBetaBlocks` | complete |  |
| `concatBlocks` | complete |  |
| `convertBudgetToEffort` | complete |  |
| `extractFirstBetaUserMessageText` | complete |  |
| `extractFirstBetaUserPromptText` | complete |  |
| `extractFirstUserMessageText` | complete |  |
| `extractFirstUserPromptText` | complete |  |
| `filterBetaThinkingBlocksInMessages` | complete |  |
| `filterThinkingBlocksInMessages` | complete |  |
| `fitAnthropicThinkingBudget` | complete |  |
| `getThinkingLevel` | complete |  |
| `hasBetaSystemRole` | complete |  |
| `hasSystemRole` | complete |  |
| `isSystemReminderText` | complete |  |
| `jsonPathIndex` | complete |  |
| `messageToMap` | complete |  |
| `normalizeToolParameters` | complete |  |
| `openaiReasoningEffort` | complete |  |
| `resolveReasoningEffort` | complete |  |
| `shortenNameIfNeeded` | complete |  |
| `stripOpenAIPromptCacheFields` | complete |  |
| `stripTextPartBreakpoints` | complete |  |
| `supportsExplicitPromptCache` | complete |  |
| `textPartText` | complete |  |
| `unionPartText` | complete |  |

</details>

<details><summary><code>internal/protocol/request</code>: 175 functions</summary>

| function | status | note |
|---|---|---|
| `CleanupOpenaiFields` | partial | 1 unmodelled site: nil comparison t26 on a value of unknown nil-ness is havoc |
| `ConvertAnthropicBetaToolsToResponses` | partial | 5 unmodelled sites: map update t15["type":string] on a map the model knows nothing about ignored |
| `ConvertAnthropicV1ToResponsesRequest` | partial | 2 unmodelled sites: > on float64: t69 is havoc (opaque-op) |
| `ConvertAnthropicV1ToolsToResponses` | partial | 5 unmodelled sites: map update t15["type":string] on a map the model knows nothing about ignored |
| `ConvertChatToOpenAIResponses` | partial | 2 unmodelled sites: > on float64: t101 is havoc (opaque-op) |
| `ConvertChatToolsToResponsesTools` | partial | 1 unmodelled site: nil comparison t16 on a value of unknown nil-ness is havoc |
| `ConvertGoogleToOpenAIRequest` | partial | 4 unmodelled sites: ConvertGoogleToOpenAIRequest: conversion float32 -> float64: assumed the zero value of float64 (not modelled) |
| `ConvertOpenAIToAnthropicTools` | partial | 1 unmodelled site: nil comparison t22 on a value of unknown nil-ness is havoc |
| `ConvertOpenAIToGoogleRequest` | partial | 6 unmodelled sites: > on float64: t19 is havoc (opaque-op) |
| `ConvertOpenAIToGoogleTools` | partial | 1 unmodelled site: nil comparison t16 on a value of unknown nil-ness is havoc |
| `ConvertRawInputToChatGPTFormat` | partial | 32 unmodelled sites: lookup in map raw the model knows nothing about is havoc |
| `ConvertResponseInputToChatGPTFormat` | partial | 11 unmodelled sites: map update t14["type":string] on a map the model knows nothing about ignored |
| `ConvertResponseToolChoiceToChatGPTFormat` | partial | 2 unmodelled sites: map update t15["type":string] on a map the model knows nothing about ignored |
| `ConvertResponseToolsToChatGPTFormat` | partial | 5 unmodelled sites: map update t18["type":string] on a map the model knows nothing about ignored |
| `ConvertResponsesToolsToAnthropicBeta` | partial | 1 unmodelled site: nil comparison t18 on a value of unknown nil-ness is havoc |
| `ConvertResponsesToolsToChatTools` | partial | 3 unmodelled sites: nil comparison t20 on a value of unknown nil-ness is havoc |
| `ExtractInstructions` | partial | 12 unmodelled sites: lookup in map raw the model knows nothing about is havoc |
| `NormalizeSchemaTypes` | partial | 1 unmodelled site: range over map t10: in insertion order; a map the model knows nothing about has no entries |
| `RepairResponsesToolCalls` | partial | 11 unmodelled sites: len of map[string]bool assumed below 16 (not modelled) |
| `applyOpenAIEffortAsThinking` | partial | 3 unmodelled sites: lookup in map t1 the model knows nothing about is havoc |
| `convertAnthropicInputSchemaToOpenAIParameters` | partial | 14 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `convertAnthropicToolViewsToGoogle` | partial | 1 unmodelled site: nil comparison t19 on a value of unknown nil-ness is havoc |
| `convertAnthropicViewAssistantToOpenAI` | partial | 1 unmodelled site: map update t19["x_thinking":string] on a map the model knows nothing about ignored |
| `convertAnthropicViewMessageToGoogle` | partial | 3 unmodelled sites: map update t122["output":string] on a map the model knows nothing about ignored |
| `convertGoogleContentToAnthropic` | partial | 3 unmodelled sites: nil comparison t52 on a value of unknown nil-ness is havoc |
| `convertGoogleContentToOpenAI` | partial | 3 unmodelled sites: nil comparison t64 on a value of unknown nil-ness is havoc |
| `imageURLToGoogleBlob` | partial | 2 unmodelled sites: imageURLToGoogleBlob: load StdEncoding: assumed a pointer to a fresh encoding/base64.Encoding with havoc fields (not ... |
| `normalizeGoogleSchemaProperties` | partial | 4 unmodelled sites: range over map props: in insertion order; a map the model knows nothing about has no entries |
| `normalizeGoogleSchemaTypes` | partial | 12 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `openAIPromptCacheKey` | partial | 2 unmodelled sites: openAIPromptCacheKey: call crypto/sha256.Sum256: assumed the zero value of [32]byte (not modelled) |
| `AnthropicParamOpt` | complete |  |
| `AnthropicParamOpt[float64]` | complete |  |
| `ConvertAnthropicBetaToGoogleRequest` | complete |  |
| `ConvertAnthropicBetaToGoogleToolChoice` | complete |  |
| `ConvertAnthropicBetaToGoogleTools` | complete |  |
| `ConvertAnthropicBetaToOpenAIRequest` | complete |  |
| `ConvertAnthropicBetaToResponsesRequest` | complete |  |
| `ConvertAnthropicBetaToV1Request` | complete |  |
| `ConvertAnthropicBetaToolChoiceToOpenAI` | complete |  |
| `ConvertAnthropicBetaToolChoiceToResponses` | complete |  |
| `ConvertAnthropicBetaToolsToOpenAI` | complete |  |
| `ConvertAnthropicBetaToolsToOpenAIWithTransformedSchema` | complete |  |
| `ConvertAnthropicToGoogleRequest` | complete |  |
| `ConvertAnthropicToGoogleToolChoice` | complete |  |
| `ConvertAnthropicToGoogleTools` | complete |  |
| `ConvertAnthropicToOpenAIRequest` | complete |  |
| `ConvertAnthropicToolChoiceToOpenAI` | complete |  |
| `ConvertAnthropicToolsToOpenAI` | complete |  |
| `ConvertAnthropicToolsToOpenAIWithTransformedSchema` | complete |  |
| `ConvertAnthropicV1ToBetaRequest` | complete |  |
| `ConvertAnthropicV1ToBetaRequestWithError` | complete |  |
| `ConvertAnthropicV1ToolChoiceToResponses` | complete |  |
| `ConvertBetaContentBlocksToString` | complete |  |
| `ConvertBetaTextBlocksToString` | complete |  |
| `ConvertChatMessagesToResponsesInput` | complete |  |
| `ConvertChatToolChoiceToResponsesToolChoice` | complete |  |
| `ConvertContentBlocksToString` | complete |  |
| `ConvertFunctionCallOutputToChatGPTFormat` | complete |  |
| `ConvertFunctionCallToChatGPTFormat` | complete |  |
| `ConvertGooglePartsToString` | complete |  |
| `ConvertGoogleToAnthropicRequest` | complete |  |
| `ConvertGoogleToolChoiceToAnthropic` | complete |  |
| `ConvertGoogleToolChoiceToOpenAI` | complete |  |
| `ConvertGoogleToolsToAnthropic` | complete |  |
| `ConvertGoogleToolsToOpenAI` | complete |  |
| `ConvertOpenAIResponsesToAnthropicBetaRequest` | complete |  |
| `ConvertOpenAIResponsesToChat` | complete |  |
| `ConvertOpenAIToAnthropicRequest` | complete |  |
| `ConvertOpenAIToAnthropicToolChoice` | complete |  |
| `ConvertOpenAIToGoogleToolChoice` | complete |  |
| `ConvertResponsesInputToMessages` | complete |  |
| `ConvertResponsesToolChoiceToAnthropicBeta` | complete |  |
| `ConvertResponsesToolChoiceToChat` | complete |  |
| `ConvertTextBlocksToString` | complete |  |
| `IsThinkingEnabled` | complete |  |
| `IsThinkingEnabledBeta` | complete |  |
| `OpenAIConfigFromResponses` | complete |  |
| `ParamOpt` | complete |  |
| `ParamOpt[float64]` | complete |  |
| `ParamOpt[github.com/openai/openai-go/v3/responses.ToolChoiceOptions]` | complete |  |
| `ParamOpt[int64]` | complete |  |
| `ParamOpt[string]` | complete |  |
| `ParseImageURLToAnthropicSource` | complete |  |
| `RequiresMaxCompletionTokens` | complete |  |
| `anthropicBetaMessagesHaveRepresentableCacheControl` | complete |  |
| `anthropicBetaMessagesHaveToolUseCacheControl` | complete |  |
| `anthropicBetaToolsHaveCacheControl` | complete |  |
| `anthropicImageToOpenAIURL` | complete |  |
| `anthropicV1MessagesHaveRepresentableCacheControl` | complete |  |
| `anthropicV1MessagesHaveToolUseCacheControl` | complete |  |
| `anthropicV1ToolsHaveCacheControl` | complete |  |
| `anthropicViewReasoningEffort` | complete |  |
| `appendBetaMessage` | complete |  |
| `applyAnthropicThinkingToResponses` | complete |  |
| `applyFirstOpenAICacheBreakpoint` | complete |  |
| `applyFirstResponsesCacheBreakpoint` | complete |  |
| `betaImageBlockToOpenAIURL` | complete |  |
| `betaTextBlockFromOpenAI` | complete |  |
| `blockHasCacheControl` | complete |  |
| `blocksHaveCacheControl` | complete |  |
| `chatTextPartsHaveCacheBreakpoint` | complete |  |
| `chatUnionPartsHaveMedia` | complete |  |
| `chatUnionTextPartsHaveCacheBreakpoint` | complete |  |
| `convertAnthropicAssistantMessageToOpenAI` | complete |  |
| `convertAnthropicBetaAssistantMessageToOpenAI` | complete |  |
| `convertAnthropicBetaUserMessageToOpenAI` | complete |  |
| `convertAnthropicToolChoiceViewToGoogle` | complete |  |
| `convertAnthropicToolChoiceViewToOpenAI` | complete |  |
| `convertAnthropicToolViewsToOpenAI` | complete |  |
| `convertAnthropicUserMessageToOpenAI` | complete |  |
| `convertAnthropicViewToGoogleRequest` | complete |  |
| `convertAnthropicViewToOpenAIRequest` | complete |  |
| `convertAnthropicViewUserToOpenAI` | complete |  |
| `convertBetaAssistantMessageToResponsesInput` | complete |  |
| `convertBetaToolResultContent` | complete |  |
| `convertBetaUserMessageToResponsesInput` | complete |  |
| `convertChatAssistantMessageToResponses` | complete |  |
| `convertChatSystemMessageToResponses` | complete |  |
| `convertChatToolMessageToResponses` | complete |  |
| `convertChatUserMessageToResponses` | complete |  |
| `convertResponsesAssistantMessageToAnthropicBeta` | complete |  |
| `convertResponsesFunctionCallOutput` | complete |  |
| `convertResponsesFunctionCallOutputToAnthropicBeta` | complete |  |
| `convertResponsesFunctionCallToAnthropicBeta` | complete |  |
| `convertResponsesInputToAnthropicBetaMessages` | complete |  |
| `convertResponsesOutputMessageToAnthropicBeta` | complete |  |
| `convertResponsesSystemInputToAnthropicBeta` | complete |  |
| `convertResponsesUserMessageToAnthropicBeta` | complete |  |
| `convertToolResultContent` | complete |  |
| `convertV1AssistantMessageToResponsesInput` | complete |  |
| `convertV1MessagesToResponsesInput` | complete |  |
| `convertV1UserMessageToResponsesInput` | complete |  |
| `createMessage` | complete |  |
| `createMessageFromResponsesContent` | complete |  |
| `dropEmptyTextBlocks` | complete |  |
| `droppedTopLevelFields` | complete |  |
| `hasAnthropicCacheControl` | complete |  |
| `hasOpenAITextCacheBreakpoint` | complete |  |
| `imageBlockToOpenAIURL` | complete |  |
| `joinTextContentParts` | complete |  |
| `joinTextContentPartsFromUnion` | complete |  |
| `markFunctionCallOutputBreakpoint` | complete |  |
| `messagesHaveThinking` | complete |  |
| `openAIImagePart` | complete |  |
| `openAIImageURLToAnthropicBetaBlock` | complete |  |
| `openAITextPart` | complete |  |
| `openAIToolMessageFromAnthropicToolResult` | complete |  |
| `orphanOutputToUserMessage` | complete |  |
| `outputMessageText` | complete |  |
| `reconcileGatewayThinking` | complete |  |
| `responseContentFromChatTextParts` | complete |  |
| `responseMessageWithContent` | complete |  |
| `responseMessageWithString` | complete |  |
| `responseOutputMessageWithText` | complete |  |
| `responseTextContentToChatTextParts` | complete |  |
| `responsesFunctionCallOutputFromToolResult` | complete |  |
| `responsesInputTextPart` | complete |  |
| `responsesOutputTextParts` | complete |  |
| `responsesTextParts` | complete |  |
| `systemBlocksHaveCacheControl` | complete |  |
| `truncateToolCallID` | complete |  |
| `viewAnthropicBetaBlock` | complete |  |
| `viewAnthropicBetaCacheControl` | complete |  |
| `viewAnthropicBetaImage` | complete |  |
| `viewAnthropicBetaMessage` | complete |  |
| `viewAnthropicBetaRequest` | complete |  |
| `viewAnthropicBetaSystem` | complete |  |
| `viewAnthropicBetaThinking` | complete |  |
| `viewAnthropicBetaToolChoice` | complete |  |
| `viewAnthropicBetaToolInputSchema` | complete |  |
| `viewAnthropicBetaTools` | complete |  |
| `viewAnthropicV1Request` | complete |  |
| `viewHasRepresentableCacheControl` | complete |  |
| `viewHasToolDefinitionCacheControl` | complete |  |
| `viewHasToolUseCacheControl` | complete |  |

</details>

<details><summary><code>internal/protocol/sse</code>: 14 functions</summary>

| function | status | note |
|---|---|---|
| `AssembleAnthropicStream` | partial | 39 unmodelled sites: lookup in map t22 the model knows nothing about is havoc |
| `AssembleGoogleStream` | partial | 13 unmodelled sites: lookup in map t19 the model knows nothing about is havoc |
| `AssembleOpenAIChatStream` | partial | 26 unmodelled sites: lookup in map t20 the model knows nothing about is havoc |
| `AssembleOpenAIResponsesStream` | partial | 24 unmodelled sites: lookup in map t20 the model knows nothing about is havoc |
| `ParseAnthropicResult` | partial | 30 unmodelled sites: lookup in map m the model knows nothing about is havoc |
| `ParseGoogleResult` | partial | 26 unmodelled sites: lookup in map m the model knows nothing about is havoc |
| `ParseOpenAIChatResult` | partial | 32 unmodelled sites: lookup in map m the model knows nothing about is havoc |
| `ParseOpenAIResponsesResult` | partial | 40 unmodelled sites: lookup in map m the model knows nothing about is havoc |
| `ReadSSELines` | partial | 4 unmodelled sites: ReadSSELines: call bufio.NewScanner: assumed a pointer to a fresh bufio.Scanner with havoc fields (not modelled) |
| `WriteSSEResponse` | partial | 13 unmodelled sites: call w: t0 is havoc (unmodeled-call) |
| `mergeAnthropicUsage` | partial | 12 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `parsedToInt` | partial | 5 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `AssembleOpenAIStream` | complete |  |
| `ParseSSEDataPayload` | complete |  |

</details>

<details><summary><code>internal/protocol/stream</code>: 184 functions</summary>

| function | status | note |
|---|---|---|
| `HandleOpenAIResponsesStreamToAnthropic` | rejected | deferred call invoke stream.Close() ignored (effects not modelled) |
| `AnthropicSingleBetaMessage` | partial | 4 unmodelled sites: type assertion t5 on an interface value the model did not build is havoc |
| `AnthropicSingleMessage` | partial | 4 unmodelled sites: type assertion t5 on an interface value the model did not build is havoc |
| `AnthropicToOpenAIStreamWithMCPHooks` | partial | 19 unmodelled sites: call (*net/http.Request).Context: t9 is havoc (unmodeled-call) |
| `AsAnthropicEvent` | partial | 1 unmodelled site: type assertion t1 on an interface value the model did not build is havoc |
| `BuildErrorEvent` | partial | 5 unmodelled sites: map update t0["type":string] on a map the model knows nothing about ignored |
| `ExtractReasoningText` | partial | 11 unmodelled sites: len of map[string]interface{} assumed below 16 (not modelled) |
| `FilterOpenAIProtocolFields` | partial | 9 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `FilterSpecialFields` | partial | 4 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `GenerateObfuscationString` | partial | 6 unmodelled sites: GenerateObfuscationString: load URLEncoding: assumed a pointer to a fresh encoding/base64.Encoding with havoc fields ... |
| `HandleAnthropic` | partial | 6 unmodelled sites: deferred call t5() ignored (effects not modelled) |
| `HandleAnthropicBeta` | partial | 6 unmodelled sites: deferred call t5() ignored (effects not modelled) |
| `HandleAnthropicBetaToOpenAIResponsesStream` | partial | 14 unmodelled sites: deferred call t5() ignored (effects not modelled) |
| `HandleAnthropicToGoogleStreamResponse` | partial | 19 unmodelled sites: call (*net/http.Request).Context: t6 is havoc (unmodeled-call) |
| `HandleGoogleToAnthropicBetaStreamResponse` | partial | 3 unmodelled sites: call (*net/http.Request).Context: t7 is havoc (unmodeled-call) |
| `HandleGoogleToAnthropicStreamResponse` | partial | 3 unmodelled sites: call (*net/http.Request).Context: t6 is havoc (unmodeled-call) |
| `HandleGoogleToOpenAIStreamResponse` | partial | 7 unmodelled sites: call (*net/http.Request).Context: t6 is havoc (unmodeled-call) |
| `HandleOpenAIChatStream` | partial | 28 unmodelled sites: deferred call (*github.com/openai/openai-go/v3/packages/ssestream.Stream[github.com/openai/openai-go/v3.ChatCompleti... |
| `HandleOpenAIChatToResponsesStream` | partial | 6 unmodelled sites: deferred call t5() ignored (effects not modelled) |
| `HandleOpenAIResponsesStream` | partial | 12 unmodelled sites: deferred call invoke t4.Close() ignored (effects not modelled) |
| `HandleOpenAIToGoogleStreamResponse` | partial | 19 unmodelled sites: call (*net/http.Request).Context: t6 is havoc (unmodeled-call) |
| `HandleResponsesToOpenAIChatStream` | partial | 6 unmodelled sites: deferred call t5() ignored (effects not modelled) |
| `MarshalAndSendErrorEvent` | partial | 1 unmodelled site: call (*net/http.Request).Context: t8 is havoc (unmodeled-call) |
| `NewChatToResponsesConverter` | partial | 2 unmodelled sites: call time.Now: t6 is havoc (unmodeled-call) |
| `NewExampleTool` | partial | 10 unmodelled sites: map update t7["type":string] on a map the model knows nothing about ignored |
| `OpenAIResponsesEvent` | partial | 7 unmodelled sites: type assertion t2 on an interface value the model did not build is havoc |
| `OpenAISSE` | partial | 7 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `OpenAISSEDone` | partial | 1 unmodelled site: call t1: t2 is havoc (unmodeled-call) |
| `PrintChunk` | partial | 5 unmodelled sites: type assertion t0 on an interface value the model did not build is havoc |
| `anthropicBetaToResponsesConverter.Next` | partial | 1 unmodelled site: load errAnthropicStreamTruncated: t31 is havoc (unmodeled-load) |
| `anthropicBetaToResponsesConverter.emitCompletionEvents` | partial | 2 unmodelled sites: range over map t17: in insertion order; a map the model knows nothing about has no entries |
| `anthropicBetaToResponsesConverter.emitContentBlockDelta` | partial | 4 unmodelled sites: lookup in map t8 the model knows nothing about is havoc |
| `anthropicBetaToResponsesConverter.emitContentBlockStart` | partial | 4 unmodelled sites: map update t11[t2] on a map the model knows nothing about ignored |
| `anthropicBetaToResponsesConverter.emitContentBlockStop` | partial | 4 unmodelled sites: lookup in map t4 the model knows nothing about is havoc |
| `anthropicToOpenAIConverter.Next` | partial | 1 unmodelled site: load errAnthropicStreamTruncated: t32 is havoc (unmodeled-load) |
| `anthropicToOpenAIConverter.processEvent` | partial | 2 unmodelled sites: nil comparison t63 on a value of unknown nil-ness is havoc |
| `anthropicToolUseBlockStartWithInput` | partial | 1 unmodelled site: nil comparison t0 on a value of unknown nil-ness is havoc |
| `chatToResponsesConverter.emitCompletionEvents` | partial | 7 unmodelled sites: len of map[int]*github.com/tingly-dev/tingly-box/internal/protocol/stream.pendingToolCallResponse assumed below 16 (n... |
| `chatToResponsesConverter.processChunk` | partial | 6 unmodelled sites: lookup in map t90 the model knows nothing about is havoc |
| `closeOpenBlock` | partial | 4 unmodelled sites: lookup in map t14 the model knows nothing about is havoc |
| `extractString` | partial | 4 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `isOpenAIChatChunkMapContent` | partial | 14 unmodelled sites: lookup in map chunk the model knows nothing about is havoc |
| `mergeMaps` | partial | 4 unmodelled sites: nil comparison t0 on a value of unknown nil-ness is havoc |
| `newAnthropicBetaToResponsesConverter` | partial | 2 unmodelled sites: call time.Now: t0 is havoc (unmodeled-call) |
| `newAnthropicToOpenAIConverter` | partial | 4 unmodelled sites: call time.Now: t6 is havoc (unmodeled-call) |
| `newResponsesToChatConverter` | partial | 4 unmodelled sites: call time.Now: t5 is havoc (unmodeled-call) |
| `openAIToAnthropicConverter.emitCloseOpenBlock` | partial | 4 unmodelled sites: lookup in map t26 the model knows nothing about is havoc |
| `openAIToAnthropicConverter.emitContentBlockStop` | partial | 1 unmodelled site: map update t9[index] on a map the model knows nothing about ignored |
| `openAIToAnthropicConverter.emitMessageDelta` | partial | 6 unmodelled sites: map update t0["stop_reason":string] on a map the model knows nothing about ignored |
| `openAIToAnthropicConverter.emitStopEvents` | partial | 8 unmodelled sites: lookup in map t22 the model knows nothing about is havoc |
| `openAIToAnthropicConverter.emitTruncatedError` | partial | 4 unmodelled sites: map update t0["type":string] on a map the model knows nothing about ignored |
| `openAIToAnthropicConverter.processChunk` | partial | 23 unmodelled sites: nil comparison t16 on a value of unknown nil-ness is havoc |
| `responsesToAnthropicConverter.Next` | partial | 4 unmodelled sites: map update t32["type":string] on a map the model knows nothing about ignored |
| `responsesToAnthropicConverter.emitContentBlockStop` | partial | 1 unmodelled site: map update t9[index] on a map the model knows nothing about ignored |
| `responsesToAnthropicConverter.emitMessageDelta` | partial | 6 unmodelled sites: map update t0["stop_reason":string] on a map the model knows nothing about ignored |
| `responsesToAnthropicConverter.emitStopEvents` | partial | 8 unmodelled sites: lookup in map t22 the model knows nothing about is havoc |
| `responsesToAnthropicConverter.finalize` | partial | 2 unmodelled sites: lookup in map t39 the model knows nothing about is havoc |
| `responsesToAnthropicConverter.processEvent` | partial | 24 unmodelled sites: map update t153[t157] on a map the model knows nothing about ignored |
| `responsesToChatConverter.emitCompletedOutput` | partial | 1 unmodelled site: len of map[int]*github.com/tingly-dev/tingly-box/internal/protocol/stream.responsesToChatToolCall assumed below 16 (no... |
| `responsesToChatConverter.processEvent` | partial | 12 unmodelled sites: map update t82[t85] on a map the model knows nothing about ignored |
| `sendAnthropicStreamEvent` | partial | 1 unmodelled site: call (*net/http.Request).Context: t8 is havoc (unmodeled-call) |
| `sendAnthropicV1MessageStop` | partial | 5 unmodelled sites: map update t4["type":string] on a map the model knows nothing about ignored |
| `sendContentBlockStop` | partial | 1 unmodelled site: map update t7[index] on a map the model knows nothing about ignored |
| `sendGoogleStreamChunk` | partial | 2 unmodelled sites: call (*net/http.Request).Context: t9 is havoc (unmodeled-call) |
| `sendMessageDelta` | partial | 8 unmodelled sites: map update t0["stop_reason":string] on a map the model knows nothing about ignored |
| `sendOpenAIStreamError` | partial | 3 unmodelled sites: map update t1["message":string] on a map the model knows nothing about ignored |
| `sendResponsesEvent` | partial | 1 unmodelled site: type assertion t0 on an interface value the model did not build is havoc |
| `sendStopEvents` | partial | 8 unmodelled sites: lookup in map t14 the model knows nothing about is havoc |
| `AnthropicEvent.RawJSON` | complete |  |
| `AnthropicToOpenAIStream` | complete |  |
| `NewAnthropicBetaToOpenAIChatConverter` | complete |  |
| `NewAnthropicBetaToOpenAIResponsesConverter` | complete |  |
| `NewOpenAIChatToAnthropicBetaConverter` | complete |  |
| `NewOpenAIChatToAnthropicV1Converter` | complete |  |
| `NewOpenAIResponsesToAnthropicConverter` | complete |  |
| `NewOpenAIResponsesToChatConverter` | complete |  |
| `PrimeResponsesStream` | complete |  |
| `RunConverter` | complete |  |
| `SendForwardingError` | complete |  |
| `SendInternalError` | complete |  |
| `SendInvalidRequestBodyError` | complete |  |
| `SendSSErrorEvent` | complete |  |
| `SendSSErrorEventJSON` | complete |  |
| `SendStreamingError` | complete |  |
| `anthropicBetaToResponsesConverter.Usage` | complete |  |
| `anthropicBetaToResponsesConverter.emitMessageStart` | complete |  |
| `anthropicBetaToResponsesConverter.nextSeq` | complete |  |
| `anthropicBetaToResponsesConverter.processEvent` | complete |  |
| `anthropicBetaToResponsesConverter.wireResponse` | complete |  |
| `anthropicInputJSONDelta` | complete |  |
| `anthropicSignatureDelta` | complete |  |
| `anthropicStopReasonToIncomplete` | complete |  |
| `anthropicTextBlockStart` | complete |  |
| `anthropicTextDelta` | complete |  |
| `anthropicThinkingBlockStart` | complete |  |
| `anthropicThinkingDelta` | complete |  |
| `anthropicToOpenAIConverter.HookErr` | complete |  |
| `anthropicToOpenAIConverter.Usage` | complete |  |
| `anthropicToOpenAIConverter.emitChunk` | complete |  |
| `anthropicToOpenAIConverter.newChunk` | complete |  |
| `anthropicToolUseBlockStart` | complete |  |
| `buildStreamState` | complete |  |
| `chatFinishReasonToIncomplete` | complete |  |
| `chatToResponsesConverter.Next` | complete |  |
| `chatToResponsesConverter.Next~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.Usage` | complete |  |
| `chatToResponsesConverter.Usage~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.emitCompletionEvents~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.emitCreated` | complete |  |
| `chatToResponsesConverter.emitCreated~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.emitTextItemAdded` | complete |  |
| `chatToResponsesConverter.emitTextItemAdded~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.nextSeq` | complete |  |
| `chatToResponsesConverter.nextSeq~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.processChunk~NewChatToResponsesConverter` | complete |  |
| `chatToResponsesConverter.wireResponse` | complete |  |
| `chatToResponsesConverter.wireResponse~NewChatToResponsesConverter` | complete |  |
| `chunkHasUsage` | complete |  |
| `firstEventReplayStream.Close` | complete |  |
| `firstEventReplayStream.Current` | complete |  |
| `firstEventReplayStream.Err` | complete |  |
| `firstEventReplayStream.Next` | complete |  |
| `isAnthropicContentDeltaEvent` | complete |  |
| `isErr` | complete |  |
| `isGoogleContentChunk` | complete |  |
| `isOpenAIChatContentChunk` | complete |  |
| `isOpenAIResponsesContentEvent` | complete |  |
| `mapOpenAIFinishReasonToAnthropic` | complete |  |
| `mapOpenAIFinishReasonToAnthropicBeta` | complete |  |
| `newAnthropicMessageStartEvent` | complete |  |
| `newOpenAIToAnthropicConverter` | complete |  |
| `newResponsesFunctionCallItem` | complete |  |
| `newResponsesMessageItem` | complete |  |
| `newResponsesToAnthropicConverter` | complete |  |
| `newStreamState` | complete |  |
| `openAIToAnthropicConverter.HookErr` | complete |  |
| `openAIToAnthropicConverter.MessageStarted` | complete |  |
| `openAIToAnthropicConverter.Next` | complete |  |
| `openAIToAnthropicConverter.Usage` | complete |  |
| `openAIToAnthropicConverter.consumeTokenCounter` | complete |  |
| `openAIToAnthropicConverter.emitAnthropic` | complete |  |
| `openAIToAnthropicConverter.emitContentBlockDelta` | complete |  |
| `openAIToAnthropicConverter.emitContentBlockStart` | complete |  |
| `openAIToAnthropicConverter.emitEnsureMessageStart` | complete |  |
| `openAIToAnthropicConverter.emitMessageStop` | complete |  |
| `openAIToAnthropicConverter.emitTerminalEvents` | complete |  |
| `openAIToAnthropicConverter.emitThinkingSignature` | complete |  |
| `openaiChatSSEWriter` | complete |  |
| `parseRawJSON` | complete |  |
| `responsesSSEWriter` | complete |  |
| `responsesToAnthropicConverter.HookErr` | complete |  |
| `responsesToAnthropicConverter.MessageStarted` | complete |  |
| `responsesToAnthropicConverter.Usage` | complete |  |
| `responsesToAnthropicConverter.emitAnthropic` | complete |  |
| `responsesToAnthropicConverter.emitContentBlockDelta` | complete |  |
| `responsesToAnthropicConverter.emitContentBlockStart` | complete |  |
| `responsesToAnthropicConverter.emitMessageStart` | complete |  |
| `responsesToAnthropicConverter.emitMessageStop` | complete |  |
| `responsesToAnthropicConverter.emitThinkingSignature` | complete |  |
| `responsesToChatConverter.Next` | complete |  |
| `responsesToChatConverter.Usage` | complete |  |
| `responsesToChatConverter.finalChunk` | complete |  |
| `responsesToChatConverter.newChunk` | complete |  |
| `responsesToChatConverter.roleChunk` | complete |  |
| `responsesToChatConverter.textChunk` | complete |  |
| `responsesToChatConverter.toolCallDeltaChunk` | complete |  |
| `responsesToChatConverter.toolCallStartChunk` | complete |  |
| `responsesToChatFinishReason` | complete |  |
| `rewriteToolCallIDForAnthropic` | complete |  |
| `sendAnthropicV1ContentBlockDelta` | complete |  |
| `sendAnthropicV1ContentBlockStart` | complete |  |
| `sendAnthropicV1ContentBlockStop` | complete |  |
| `sendAnthropicV1MessageStart` | complete |  |
| `sendContentBlockDelta` | complete |  |
| `sendContentBlockStart` | complete |  |
| `sendMessageStart` | complete |  |
| `sendMessageStop` | complete |  |
| `sendOpenAIStreamChunkForce` | complete |  |
| `sendResponsesErrorEvent` | complete |  |
| `sendThinkingSignature` | complete |  |
| `setAnthropicSSEHeaders` | complete |  |
| `truncateToolCallID` | complete |  |
| `wireStr` | complete |  |
| `writeSSEChunk` | complete |  |

</details>

<details><summary><code>internal/protocol/thinking</code>: 6 functions</summary>

| function | status | note |
|---|---|---|
| `EffortFromBudget` | partial | 5 unmodelled sites: lookup in map t0 the model knows nothing about is havoc |
| `thinkingSampling` | partial | 2 unmodelled sites: != on float64: t7 is havoc (opaque-op) |
| `AnthropicEffort` | complete |  |
| `DisableBetaForUnsignedToolTurn` | complete |  |
| `ReconcileBetaWithRequest` | complete |  |
| `betaToolTurnWithoutThinking` | complete |  |

</details>

<details><summary><code>internal/protocol/token</code>: 11 functions</summary>

| function | status | note |
|---|---|---|
| `StreamTokenCounter.ConsumeOpenAIChunk` | partial | 2 unmodelled sites: call to (*sync.Mutex).Lock ignored (no result, effects not modelled) |
| `StreamTokenCounter.GetCounts` | partial | 2 unmodelled sites: call to (*sync.Mutex).Lock ignored (no result, effects not modelled) |
| `StreamTokenCounter.GetUpstreamDetails` | partial | 2 unmodelled sites: call to (*sync.Mutex).Lock ignored (no result, effects not modelled) |
| `StreamTokenCounter.SetInputTokens` | partial | 2 unmodelled sites: call to (*sync.Mutex).Lock ignored (no result, effects not modelled) |
| `estimateInputTokensWith` | partial | 12 unmodelled sites: type assertion t20 on an interface value the model did not build is havoc |
| `getCodec` | partial | 4 unmodelled sites: call (*sync.Map).Load: t1 is havoc (unmodeled-call) |
| `EstimateInputTokensSimple` | complete |  |
| `EstimateOutputTokens` | complete |  |
| `NewStreamTokenCounter` | complete |  |
| `StreamTokenCounter.countTokens` | complete |  |
| `StreamTokenCounter.flushPendingLocked` | complete |  |

</details>

<details><summary><code>internal/protocol/usage</code>: 19 functions</summary>

| function | status | note |
|---|---|---|
| `AnthropicAccumulator.Consume` | partial | 1 unmodelled site: call strings.Clone: t5 is havoc (unmodeled-call) |
| `AnthropicAccumulator.ConsumeBeta` | complete |  |
| `AnthropicAccumulator.ConsumeBeta~NewAnthropicAccumulator` | complete |  |
| `AnthropicAccumulator.Consume~NewAnthropicAccumulator` | complete |  |
| `AnthropicAccumulator.HasUsage` | complete |  |
| `AnthropicAccumulator.HasUsage~NewAnthropicAccumulator` | complete |  |
| `AnthropicAccumulator.Result` | complete |  |
| `AnthropicAccumulator.Result~NewAnthropicAccumulator` | complete |  |
| `AnthropicAccumulator.consumeRaw` | complete |  |
| `AnthropicAccumulator.consumeRaw~NewAnthropicAccumulator` | complete |  |
| `ChatUsage` | complete |  |
| `FromAnthropicBetaMessage` | complete |  |
| `FromAnthropicMessage` | complete |  |
| `FromOpenAIChatCompletion` | complete |  |
| `FromOpenAIResponses` | complete |  |
| `NewAnthropicAccumulator` | complete |  |
| `ToChatStreamUsageWire` | complete |  |
| `ToChatUsageWire` | complete |  |
| `ToResponsesUsageWire` | complete |  |

</details>

<details><summary><code>internal/protocol/wire</code>: 21 functions</summary>

| function | status | note |
|---|---|---|
| `ResponsesContentPartWire.MarshalJSON` | partial | 6 unmodelled sites: map update t1["type":string] on a map the model knows nothing about ignored |
| `ChatCompletionMessageWire.MarshalJSON` | complete |  |
| `ChatCompletionWire.ToMap` | complete |  |
| `NewResponsesStreamErrorEvent` | complete |  |
| `ResponsesCompletedEvent.EventType` | complete |  |
| `ResponsesContentPartAddedEvent.EventType` | complete |  |
| `ResponsesContentPartDoneEvent.EventType` | complete |  |
| `ResponsesCreatedEvent.EventType` | complete |  |
| `ResponsesFunctionCallArgumentsDeltaEvent.EventType` | complete |  |
| `ResponsesFunctionCallArgumentsDoneEvent.EventType` | complete |  |
| `ResponsesInProgressEvent.EventType` | complete |  |
| `ResponsesIncompleteEvent.EventType` | complete |  |
| `ResponsesOutputItemAddedEvent.EventType` | complete |  |
| `ResponsesOutputItemDoneEvent.EventType` | complete |  |
| `ResponsesOutputTextDeltaEvent.EventType` | complete |  |
| `ResponsesOutputTextDeltaEvent.MarshalJSON` | complete |  |
| `ResponsesOutputTextDoneEvent.EventType` | complete |  |
| `ResponsesOutputTextDoneEvent.MarshalJSON` | complete |  |
| `ResponsesStreamErrorEvent.EventType` | complete |  |
| `ResponsesStreamErrorEvent.EventType~NewResponsesStreamErrorEvent` | complete |  |
| `ResponsesWireResponse.ToMap` | complete |  |

</details>

<details><summary><code>internal/typ</code>: 1 functions</summary>

| function | status | note |
|---|---|---|
| `ThinkingEffortFromBudget` | complete |  |

</details>

## Reproduce in one command

Needs Go and git, nothing from go2lean:

```sh
git clone https://github.com/andreasfoo/go2lean-tingly-box
cd go2lean-tingly-box
scripts/reproduce.sh                       # clones tingly-box main into a temp dir
scripts/reproduce.sh ~/src/tingly-box      # or use a checkout (left as it was)
scripts/reproduce.sh ~/src/tingly-box v1.2.3   # or check out a ref first
```

It copies the tests of `repro/` (one small `go test` per finding, written
as the behaviour a client expects: a failing test is a finding that still
reproduces) into the checkout, runs them, removes them, and prints:

```
test                                  result       finding
ToolCallIDsPairAfterConversion        reproduces   issue-1924-tool-call-ids (tracked #1924)
...
StreamedRefusalReachesChatClient      reproduces   responses-chat-stream-refusal (new)
```

`fixed` means the test passes on that tingly-box. The tests can be taken
into tingly-box as regression tests as they are.

## Running go2lean

go2lean needs Go 1.26 and Lean 4 (installed by `elan`, which picks the
version pinned in go2lean's `lean/lean-toolchain`):

```sh
curl https://raw.githubusercontent.com/leanprover/elan/master/elan-init.sh -sSf | sh
git clone https://github.com/andreasfoo/go2lean
git clone --recurse-submodules https://github.com/tingly-dev/tingly-box
git clone https://github.com/andreasfoo/go2lean-tingly-box
```

The first run builds the Lean side (the engines and one Lean module per
compiled Go package, a few minutes; later runs rebuild only what changed).
Everything below runs from this repository; go2lean writes a workspace of
models, properties, counterexamples and verdicts.

**Check the properties** of the protocol packages (about 45 minutes):

```sh
scripts/update.sh ../go2lean ../tingly-box main /tmp/tb-main
```

It checks the ref out, installs `specs/` into the checkout as
`go2lean_specs_test.go` files, makes sure they still compile, hunts every
spec, verifies each counterexample with `go test`, and writes
`/tmp/tb-main/report.md`: which known finding still reproduces, and every
spec's verdict. Run it whenever tingly-box moves; a spec that no longer
compiles stops it with `go vet`'s message.

**Hunt a few properties** (by a pattern over property ids):

```sh
scripts/install-specs.sh ../tingly-box
scripts/round.sh ../go2lean ../tingly-box /tmp/ws 'roundtrip' /tmp/round.log
```

`BUDGET` (seconds per property, default 120) and `JOBS` (default 2) tune it.

**A full pass** over every package and function (`no_panic` everywhere plus
the installed specs; several hours, resumable):

```sh
scripts/full-pass.sh ../go2lean ../tingly-box out-pass /tmp/pass.log   # rerun to resume
go run ../go2lean/cmd/go2lean summary -workspace ../go2lean/out-pass
```

`passes/` has the result of every pass so far and a triage of the last.

**Write properties with the agent** (an LLM proposes specs, go2lean checks
them):

```sh
cd ../go2lean
go run ./cmd/go2lean agent -deps -guidance ../go2lean-tingly-box/guidance/guidance.md \
  -module ../tingly-box ./internal/protocol/request
```

`ANTHROPIC_API_KEY` must be set; `-base-url` and `-model` point it at any
Anthropic-compatible endpoint.

## Layout

| path | contents |
|---|---|
| `findings/` | one page per finding; `index.tsv` maps each to the specs that show it |
| `repro/` | one plain `go test` per finding, used by `scripts/reproduce.sh` |
| `specs/` | the properties (spec-in-Go), per tingly-box package |
| `guidance/`, `agent-runs/` | guidance given to go2lean's agent, and what it wrote |
| `poc/` | the tests go2lean generated for the confirmed counterexamples, per round |
| `passes/` | full passes: counts per pass, and the triage of the last |
| `hunting.md` | the rounds in order: properties, engine output, verdicts, model changes |
| `scripts/` | `reproduce.sh`, `update.sh`, `round.sh`, `full-pass.sh`, `install-specs.sh` |
