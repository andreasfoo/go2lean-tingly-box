# Finding 71: Responses -> Chat streams lose the `response.failed` message and take the `error` code from `param` (new, not reported)

Severity: **low** (see [the findings table](README.md)). Related to
[finding 20](upstream-errors-not-translated.md) (errors before the stream
starts); this one is about errors inside the stream.

`responsesToChatConverter.processEvent`
(`internal/protocol/stream/openai_responses_to_chat_converter.go:106-228`):

- has no case for `response.failed`. The stream then ends without a
  terminal event and `Next` returns "responses stream ended without a
  terminal event" (`:85`); the upstream's error is lost.
- turns an `error` event into a Chat stream error with `Code: evt.Param`
  (`:217-224`), so the code is the event's `param` (usually null), not its
  `code`.

Path: a Chat client with a Responses provider, streaming:
`protocolserver/protocol_cross.go:83`, `stage/openaibridge/responses_stream.go:24`.

Trigger:
- `{"type":"response.failed","response":{...,"status":"failed","error":{"code":"server_error","message":"The model produced invalid content."}}}`;
- `{"type":"error","code":"rate_limit_exceeded","message":"Rate limit reached for gpt-5.","param":null}`.

Observed (openai-go's stream `Err()` on the client):
- `{"message":"responses stream ended without a terminal event","type":"stream_error"}`;
- `{"message":"Rate limit reached for gpt-5.","type":"error","code":""}`.

Expected: the upstream message "The model produced invalid content.", and
the code `rate_limit_exceeded` (which clients use to decide whether to
retry).

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md).

Repro: `TestGo2LeanRepro_ResponsesToChatStreamFailedKeepsUpstreamError` and
`TestGo2LeanRepro_ResponsesToChatStreamErrorKeepsCode`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`).
