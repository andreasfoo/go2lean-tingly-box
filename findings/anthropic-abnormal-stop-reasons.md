# Anthropic `model_context_window_exceeded` and `refusal` reported as normal completions (new, not reported)

Path: Chat or Responses client -> Anthropic provider (non-streaming Chat and
Responses, streaming Responses).

Where (tingly-box `b788a3b`):

- `internal/protocol/nonstream/anthropic.go:50-56`
  (`ConvertAnthropicBetaToOpenAIChat`): `finishReason := "stop"`, changed
  only for `tool_use` (`tool_calls`) and `max_tokens` (`length`).
- `internal/protocol/nonstream/openai_to_responses.go:203`
  (`anthropicStopReasonToResponsesStatus`, used by
  `ConvertAnthropicBetaToResponsesWire`): `max_tokens` is `incomplete`,
  everything else `completed`.
- `internal/protocol/stream/anthropic_beta_to_openai_responses_converter.go:397`
  (`anthropicStopReasonToIncomplete`): the same, for the stream.

Trigger: an Anthropic response with `stop_reason`
`model_context_window_exceeded` (Claude Sonnet 4.5 and later end a turn
that ran into the context window with it) or `refusal` (the streaming
classifiers stopped the answer).

Observed: Chat `finish_reason: "stop"`; Responses `status: "completed"`
(stream: the terminal event is `response.completed`). `max_tokens` is
right (`length`, `incomplete` / `max_output_tokens`).

Expected: not a normal completion. Chat: `length` for the context window,
`content_filter` for a refusal. Responses: `status: "incomplete"` with
`incomplete_details.reason` `max_output_tokens` and `content_filter`
respectively (the two reasons the Responses API has).

Realism: both stop reasons are documented and current. A client told
`stop`/`completed` takes a truncated or refused answer as final: Codex (a
Responses client) does not compact or retry on a context-window stop, and a
Chat client shows half an answer as the whole. The Anthropic -> Chat stream
has the same shape of mapping (its `max_tokens` case is #1923); it was not
re-tested for these two reasons.

Status: **new** (tracker search 2026-10-20 for `stop_reason`, `refusal`,
`context_window`: nothing about this; #1923 is `max_tokens` on the
Anthropic -> Chat stream).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_AbnormalStopReasonsNotCompleted`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_hunt2_test.go`, Chat and
Responses, both stop reasons) and
`TestGo2LeanRepro_ContextWindowExceededStreamIncomplete`
(`repro/internal/protocol/stream/zz_go2lean_repro_hunt2_test.go`, Responses
stream).

Fix direction: one shared table from Anthropic stop reasons to Chat finish
reasons and Responses status/reason, with `stop` / `completed` only for
`end_turn`, `stop_sequence` (and `tool_use` as today).

## General property

For every Anthropic message, the converted Chat `finish_reason` is `stop`
and the converted Responses `status` is `completed` only when the Anthropic
`stop_reason` denotes a normal end of turn (`end_turn`, `stop_sequence`,
and `tool_use` as `tool_calls`); every stop reason that cut the answer short
maps to a non-normal finish.

Target functions: `nonstream.ConvertAnthropicBetaToOpenAIChat`,
`nonstream.ConvertAnthropicBetaToResponsesWire`, and
`(*anthropicBetaToResponsesConverter).Next` (via
`NewAnthropicBetaToOpenAIResponsesConverter`).
