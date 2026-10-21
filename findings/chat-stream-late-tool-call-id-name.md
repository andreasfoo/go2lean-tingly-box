# Finding 72: Chat -> Responses and Chat -> Anthropic streams take a tool call's id and name only from its first delta (new, not reported)

Severity: **low** (see [the findings table](README.md)); no provider is
known to send this shape. Related to
[finding 17](chat-tool-call-empty-id.md) (calls that never get an id).

`chatToResponsesConverter.processChunk`
(`internal/protocol/stream/openai_chat_to_responses_converter.go:143-170`)
and the Chat -> Anthropic converter
(`stream/openai_to_anthropic_converter.go:249-266`) emit the
`function_call` item / `tool_use` block start as soon as a tool-call index
first appears, with that delta's id and name. An id or name that arrives in
a later delta for the same index only updates internal state; the client
already has the item with an empty name (and, on the Responses path, a
fabricated `call_id`).

Path: a Responses client (Codex) or an Anthropic client with a Chat
provider, streaming: `protocolserver/protocol_cross.go:54`,
`stage/responsesbridge/chat_stream.go:24`, and the Chat -> Anthropic
stream.

Trigger: Chat chunks
`{"tool_calls":[{"index":0,"type":"function","function":{"arguments":""}}]}`,
then
`{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]}`,
then `finish_reason: "tool_calls"`.

Observed: the Responses client's `function_call` has name `""` and a
generated `call_id` (`call_74c0…`); the Anthropic client's `tool_use` has
name `""` and id `""`.

Expected: `get_weather` / `call_1`, which is what openai-go's
`ChatCompletionAccumulator` makes of the same chunks.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md).

Repro: `TestGo2LeanRepro_ChatStreamLateToolCallIDAndName`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`), Responses
and Anthropic Beta clients.
