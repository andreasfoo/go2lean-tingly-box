# Finding 48: Chat finish_reason "stop" with tool calls becomes Anthropic end_turn (found by go2lean, blind specs; new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Anthropic client -> Chat provider, response conversion, non-stream
and stream.

Where (tingly-box `b788a3b`):

- non-stream: `ConvertOpenAIChatToAnthropicBeta`
  (`internal/protocol/nonstream/openai_to_anthropic.go:116-168`) starts
  from `end_turn` (`:126`) and derives the stop reason from
  `finish_reason` only (`:150-154`), although it has just turned the tool
  calls into tool_use blocks (`:143-149`). Live via the Anthropic bridge
  (`internal/protocol/stage/anthropicbridge/bridge.go:119`).
- stream, by reading and confirmed by the stream test: the mappers
  `mapOpenAIFinishReasonToAnthropic`
  (`internal/protocol/stream/openai_to_anthropic.go:32-46`) and
  `mapOpenAIFinishReasonToAnthropicBeta`
  (`internal/protocol/stream/openai_to_anthropic_beta.go:15-29`) map
  `stop` to `end_turn` whatever was streamed
  (`internal/protocol/stream/openai_to_anthropic_converter.go:341`).
- the repo's own `internal/protocol/stream/anthropic_single_message.go:59-61`
  does it right: `tool_use` whenever a tool_use block was sent.

Trigger: a Chat completion with `tool_calls:[get_weather]` and
`finish_reason: "stop"` (or the same as a stream).

Observed: `content: [tool_use]`, `stop_reason: "end_turn"` (non-stream and
both stream mappers).

Expected: `stop_reason: "tool_use"` whenever the message holds a tool_use
block.

Realism: medium. Several OpenAI-compatible servers report `stop` for a
tool call: it is reported for Gemini's OpenAI-compatible endpoint, which
tingly-box uses for the Google OAuth issuer
(`internal/server/module/oauth/handler.go:1235-1237`), and for some vLLM
and Ollama versions (not re-verified here). Anthropic clients and SDK
loops that branch on `stop_reason == "tool_use"` end the turn instead of
running the tool. Not covered by `anthropic-abnormal-stop-reasons.md` (#16).

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `end_turn` with `tool`, `stop_reason` with `tool_use`: nothing).
Reproduces on main `bc460f6`.

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)): `SpecConvertOpenAIChatToAnthropicBetaStopReason`
was refuted, `go test` confirmed the counterexample, and triage found no
earlier finding for it. The stream half was found by reading the mappers;
the repro tests were written afterwards.

Evidence: `TestGo2LeanRepro_ChatStopWithToolCallsIsToolUse`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_blind_test.go`) and
`TestGo2LeanRepro_StreamChatStopWithToolCallsIsToolUse`
(`repro/internal/protocol/stream/zz_go2lean_repro_blind_test.go`).

## General property

For every Chat completion (or chunk stream) converted to Anthropic, if the
message holds a tool_use block then its stop reason is `tool_use` (unless
it is `max_tokens`).

Target functions: `nonstream.ConvertOpenAIChatToAnthropicBeta` and the
stream converter with `stream.mapOpenAIFinishReasonToAnthropic` /
`stream.mapOpenAIFinishReasonToAnthropicBeta`.
