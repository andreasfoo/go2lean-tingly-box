# Finding 69: Chat -> Responses streams drop the refusal (new, not reported)

Severity: **low** (see [the findings table](README.md)). The stream
version of [finding 55](chat-responses-refusal-dropped.md) (non-stream).

`chatToResponsesConverter.processChunk`
(`internal/protocol/stream/openai_chat_to_responses_converter.go:106-208`)
has no handling for `delta.refusal`: no message item and no `refusal`
content part are emitted for it.

Path: a Responses client (Codex) with a Chat provider, streaming:
`protocolserver/protocol_cross.go:54`, `stage/responsesbridge/chat_stream.go:24`.
OpenAI models refuse this way, mostly with structured outputs; Chat-only
providers seldom send `refusal`.

Trigger: Chat chunks
`{"delta":{"role":"assistant","content":null,"refusal":""}}`,
`{"delta":{"refusal":"I can't help with that."}}`, then
`finish_reason: "stop"`.

Observed: the Responses client's `response.completed` output is `[]`, an
empty completed answer with no reason.

Expected: a message item with a
`{"type":"refusal","refusal":"I can't help with that."}` content part (the
refusal openai-go's accumulator gives for the same chunks).

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md).

Repro: `TestGo2LeanRepro_ChatToResponsesStreamKeepsRefusal`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`).
