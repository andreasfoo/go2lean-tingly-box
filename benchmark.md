# Benchmark: can go2lean find the bugs found by hand?

Findings 6-11 (see [findings/](findings/)) were found by reading code and
checking the bug classes of similar gateways
([hunting-similar-tools.md](hunting-similar-tools.md)), not by go2lean. They
are used as a benchmark for the tool: for each, the general property a
reviewer would state is written as a spec (no trigger input in it), go2lean
searches blind, and `go test` confirms what it finds. A miss names a gap in
go2lean, which is then closed (go2lean `docs/design.md` §9c).

| finding | spec (in `specs/`) | go2lean | gap / change |
|---|---|---|---|
| [chat-request-params](findings/chat-request-params.md), sampling and stop | `SpecConvertAnthropicBetaToOpenAIRequestKeepsSampling` | found at trial 0, confirmed | — |
| [chat-request-params](findings/chat-request-params.md), tool_choice | `SpecConvertAnthropicBetaToOpenAIRequestKeepsToolChoice` | found at trial 17, confirmed | — |
| [responses-done-only-arguments](findings/responses-done-only-arguments.md) | `SpecResponsesToChatConverterProcessEventArgumentsAnySchedule` | found at trial 610, confirmed | the event schedule is an input of the spec (the earlier stream specs replayed one fixed schedule and could not see this) |
| [chat-anthropic-reasoning-null](findings/chat-anthropic-reasoning-null.md), stream | `SpecOpenAIToAnthropicConverterProcessChunkOneOpenBlock` | pending | needed JSON metadata of SDK types and `json.Unmarshal` (both added) |
| [chat-anthropic-reasoning-null](findings/chat-anthropic-reasoning-null.md), non-stream | — | out of reach | struct JSON encoding / decoding not modelled |
| [usage-nonstream-cached-input](findings/usage-nonstream-cached-input.md) | — | out of reach | inline in an HTTP handler |
| [env-credentials-upstream](findings/env-credentials-upstream.md) | — | out of reach | environment read inside the SDK |
| [recording-client-credentials](findings/recording-client-credentials.md) | — | not a property yet | — |

Run with `scripts/round.sh <go2lean> <tingly-box> <workspace> '<spec pattern>' <log>`.
