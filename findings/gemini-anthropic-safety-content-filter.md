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

Observed (a blocked candidate as Gemini sends it, with no content:
`{"candidates":[{"finishReason":"SAFETY","index":0,"safetyRatings":[{"category":"HARM_CATEGORY_DANGEROUS_CONTENT","probability":"HIGH","blocked":true}]}],…}`;
in the stream, after a chunk of answer text): non-stream
`"stop_reason":"content_filter"`; stream
`{"type":"message_delta","delta":{"stop_reason":"content_filter",…}}`.
The same response gives `"stop_reason":"refusal"` to a Beta client.

Expected: `"stop_reason":"refusal"`, as on the Beta path.

Realism: low. Safety blocks of the answer are rare with Gemini's default
safety settings, and the v1 path is
taken only by clients that call `/v1/messages` without `?beta=true`
(`internal/protocolserver/anthropic_message.go:25`); Claude Code and the
SDKs' `beta.messages` use the Beta path, which is right. The official
Anthropic SDKs do not reject the unknown value (the Go SDK types
`stop_reason` as a string, the Python and TypeScript SDKs do not validate
response enums), so the effect is that a client that branches on
`stop_reason == "refusal"` misses the block and treats the empty or
cut-short answer as an ordinary one. Not covered by
`anthropic-abnormal-stop-reasons.md` (#16), which is about Anthropic ->
Chat and Responses.

Related, not counted here: Gemini's other block reasons (`BLOCKLIST`,
`PROHIBITED_CONTENT`, `SPII`, `IMAGE_SAFETY`, …) become `end_turn` on
both the v1 and the Beta path (`:205-210`, `:299-304`).

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
