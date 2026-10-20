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
| [chat-anthropic-reasoning-null](findings/chat-anthropic-reasoning-null.md), non-stream | — | out of reach | the converter marshals a struct and unmarshals it into an SDK type: struct JSON encoding / decoding is not modelled |
| [usage-nonstream-cached-input](findings/usage-nonstream-cached-input.md) | — | out of reach | inline in an HTTP handler |
| [env-credentials-upstream](findings/env-credentials-upstream.md) | — | out of reach | the environment is read inside the SDK |
| [recording-client-credentials](findings/recording-client-credentials.md) | — | no property yet | — |

**5 of 5 reachable cases confirmed** (`scripts/benchmark.sh`, 2026-10-20).
Misses on the way were sorted into three kinds: the spec (a fixed event
schedule; chunks skipped for a random finish reason), go2lean (inputs it
could not generate, calls it did not model, harness renderings that
changed the value), and reach (code the function-level search does not
get to).

Run with `scripts/benchmark.sh <go2lean> <tingly-box> <workspace>` (`REUSE=1` hunts again in a workspace compiled before); the cases are `benchmark.tsv`.
