# Finding 68: Chat -> Responses streams drop `reasoning_content` (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

`chatToResponsesConverter.processChunk`
(`internal/protocol/stream/openai_chat_to_responses_converter.go:106-208`)
handles `delta.content` and `delta.tool_calls` only; nothing reads
`reasoning_content` (the DeepSeek-style reasoning field, which tingly-box
itself reads on the Chat -> Anthropic path, `stream/openai_helper.go:77`).
No reasoning item is emitted. The non-stream converter
(`nonstream/openai_to_responses.go`) does not read it either (by reading;
not run).

Path: a Responses client (Codex) with a Chat provider that streams
reasoning (DeepSeek, Kimi and similar), streaming:
`protocolserver/protocol_cross.go:54`, `stage/responsesbridge/chat_stream.go:24`.

Trigger: Chat chunks
`{"delta":{"role":"assistant","content":null,"reasoning_content":"The user greets me."}}`,
`{"delta":{"content":"Hello!"}}`, then `finish_reason: "stop"`.

Observed: the Responses client's `response.completed` output is
`[message]`; the reasoning is gone.

Expected: a `reasoning` item carrying "The user greets me." before the
message, as the provider sent it.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md): openai-go decodes the
provider's chunks and the Responses events tingly-box makes of them.

Repro: `TestGo2LeanRepro_ChatToResponsesStreamKeepsReasoning`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`).
