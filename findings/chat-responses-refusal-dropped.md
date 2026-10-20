# Finding 55: Chat -> Responses (non-stream): the refusal dropped (found by go2lean, blind specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Responses client -> Chat provider, response conversion, non-stream.

Where (tingly-box `b788a3b`):
`internal/protocol/nonstream/openai_to_responses.go:40-105`
(`ConvertChatToResponsesWire`) reads only `Message.Content` (`:49`) and
emits a message item only when it is non-empty (`:57-71`);
`Message.Refusal` is never read, so no `refusal` content part is emitted.
Callers: `internal/protocol/stage/responsesbridge/chat.go:81` and, through
`HandleOpenAIChatToResponses`, `internal/protocolserver/protocol_cross.go:34`.

Trigger: a Chat completion
`{"message":{"role":"assistant","content":null,"refusal":"I can't help with that."},"finish_reason":"stop"}`.

Observed: `"output":[]`, `"status":"completed"`.

Expected: a message item with a `{"type":"refusal","refusal":"I can't help with that."}`
content part.

Realism: low. OpenAI models refuse this way, mostly with structured
outputs; Chat-only providers seldom emit `refusal`. The client sees an
empty completed answer and no reason. `responses-anthropic-nonstream-refusal.md`
(#26) is the Responses -> Anthropic analogue,
`responses-chat-stream-refusal.md` (#5) the Responses -> Chat stream.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)): `SpecConvertChatToResponsesWireConservation`
was refuted, `go test` confirmed the counterexample, and triage found no
earlier finding for this direction. The repro test was written afterwards
with a JSON-parsed completion.

Evidence: `TestGo2LeanRepro_ChatRefusalReachesResponsesClient`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_blind_test.go`).

## General property

For every Chat completion, the first choice's content, refusal and tool
calls all appear in the Responses output (content as `output_text`,
refusal as a `refusal` part, each tool call as a `function_call` item).

Target function: `nonstream.ConvertChatToResponsesWire`.
