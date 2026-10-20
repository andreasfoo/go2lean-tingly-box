# Finding 59: Gemini SAFETY becomes stop_reason "content_filter" for Anthropic v1 clients (found by go2lean, M2 agent specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Anthropic v1 client -> Google-style provider, response conversion,
non-stream and stream.

Where (tingly-box `b788a3b`):

- `internal/protocol/nonstream/google_to_any.go:197-214`
  (`MapGoogleFinishReasonToAnthropic`) maps `SAFETY` to `"content_filter"`
  (`:203-204`), which is not an Anthropic stop reason (those are
  `end_turn`, `max_tokens`, `stop_sequence`, `tool_use`, `pause_turn`,
  `refusal`, `model_context_window_exceeded`). Its Beta twin
  `MapGoogleFinishReasonToAnthropicBeta` (`:291-309`) maps it to `refusal`.
- non-stream: `HandleGoogleToAnthropic` (`:123-195`, stop reason at
  `:170`), called for Anthropic v1 clients at
  `internal/protocolserver/protocol_dispatch.go:285`.
- stream, by reading and confirmed by the stream test:
  `HandleGoogleToAnthropicStreamResponse`
  (`internal/protocol/stream/google_to_any.go:211`, stop reason at `:324`),
  called at `internal/protocolserver/protocol_dispatch.go:243`.

Trigger: Gemini blocks the answer for safety (`finishReason: "SAFETY"`)
and the client speaks the Anthropic v1 Messages API.

Observed: non-stream `"stop_reason":"content_filter"`; stream
`{"type":"message_delta","delta":{"stop_reason":"content_filter",…}}`.
The same response gives `"stop_reason":"refusal"` to a Beta client.

Expected: `"stop_reason":"refusal"`, as on the Beta path.

Realism: low. Safety blocks are rare and the path (Anthropic v1 client,
Gemini provider) is not the most common one. A client that branches on
`stop_reason == "refusal"` misses the block and shows a cut-short answer
as complete; a client that validates the stop reason against the
Anthropic enum rejects the response (not checked against specific
clients). Not covered by `anthropic-abnormal-stop-reasons.md` (#16), which
is about Anthropic -> Chat and Responses.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from the M2 agent's own specs (see the
[provenance note](README.md)):
`SpecMapGoogleFinishReasonToAnthropicAgreesWithBeta` was refuted
(`SAFETY`), `go test` confirmed the counterexample, and triage found no
earlier finding for it. The stream half was found by reading the caller;
the repro tests were written afterwards.

Evidence: `TestGo2LeanRepro_GeminiSafetyIsRefusalForAnthropicV1`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_m2_test.go`) and
`TestGo2LeanRepro_StreamGeminiSafetyIsRefusalForAnthropicV1`
(`repro/internal/protocol/stream/zz_go2lean_repro_m2_test.go`).

## General property

For every Gemini finish reason, the Anthropic v1 and Beta converters give
the same stop reason, and it is one of the Anthropic stop reasons.

Target functions: `nonstream.MapGoogleFinishReasonToAnthropic` (through
`nonstream.HandleGoogleToAnthropic` and
`stream.HandleGoogleToAnthropicStreamResponse`).
