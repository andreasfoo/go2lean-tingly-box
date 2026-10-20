# Responses -> Anthropic: tool_choice "none" becomes auto (found by the go2lean M2 agent; new, not reported)

Path: Responses client -> Anthropic provider (request conversion; streaming
or not).

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_responses_to_anthropic.go:416-422`
(`ConvertResponsesToolChoiceToAnthropicBeta`): mode `"none"` returns
`{"type":"auto"}`. The comment says "without tools defined, auto won't call
tools", but the function is called only when the request has tools, and the
caller converts the tools with it
(`ConvertOpenAIResponsesToAnthropicBetaRequest`, `:84-90`), so the tools are
always defined when it runs.

Trigger: a Responses request with tools and `tool_choice: "none"`.

Observed: the Anthropic request carries the tools and
`tool_choice: {"type":"auto"}`.

Expected: `{"type":"none"}` (Anthropic has the mode; the SDK's
`BetaToolChoiceNoneParam`).

Realism: medium. Clients set `"none"` to forbid tool calls for one turn
while keeping the tool list (and the prompt cache) unchanged, for instance
to have the model summarise or answer in text. As `auto` Claude may call a
tool the client ruled out, which an agent loop then executes or fails on.
The same mistake on other paths: `chat-anthropic-request-coverage.md`
(Chat -> Anthropic), `chat-request-params.md` (Anthropic -> Chat).

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by the go2lean M2 agent with its own spec, MiniMax-M3,
not by hand; the repro test was written afterwards and confirmed with
`go test`.

Evidence: `TestGo2LeanRepro_ResponsesToolChoiceNoneReachesAnthropic`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Responses request with tools, the Anthropic `tool_choice` keeps
the mode: `auto` -> `auto`, `required` -> `any`, `none` -> `none`, a named
function -> `tool` with that name.

Target function: `request.ConvertResponsesToolChoiceToAnthropicBeta` (via
`request.ConvertOpenAIResponsesToAnthropicBetaRequest`).
