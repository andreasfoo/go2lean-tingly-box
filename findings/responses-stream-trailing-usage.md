# Finding 13: Responses clients get no usage when streaming from a Chat provider (new, not reported)

Path: Responses client -> Chat provider, streaming (`/tingly/openai/v1/responses`
with `stream: true`, OpenAI-style provider in Chat mode).

Where (tingly-box `b788a3b`):

- `internal/protocol/stream/openai_chat_to_responses_converter.go:192-206`
  (`processChunk`): on the chunk carrying `finish_reason` the converter calls
  `emitCompletionEvents` (`:211`), which builds `response.completed` with
  `wireResponse` (`:372`, `Usage: c.Usage()`) right away. With
  `stream_options.include_usage` an OpenAI Chat stream sends usage in a
  *later* chunk with `"choices": []`; `processChunk` (`:114`) still reads it,
  but `response.completed` is already out (`completedSent`).
- The Responses -> Chat request (`request.ConvertOpenAIResponsesToChat`, used
  by `transform/base.go:105` and `stage/responsesbridge/chat.go:47`) never
  sets `stream_options.include_usage`; only the Anthropic -> Chat request
  does (`request/anthropic_view.go:349`). So an OpenAI upstream sends no
  usage at all on this path.

Trigger: any streaming Responses request routed to a Chat provider.

Observed: `response.completed` carries `input_tokens: 0` and an estimated
`output_tokens` (text length / 4). For upstream usage `prompt_tokens` 1200,
`completion_tokens` 7, the client gets
`"usage":{"input_tokens":0,…,"output_tokens":1,…,"total_tokens":1}`, and the
upstream request is `{"messages":[…],"model":"gpt-4o","max_tokens":8192,"stream":true}`.

Expected: the request asks for usage (`stream_options.include_usage: true`),
and `response.completed` is emitted after the upstream stream ends, carrying
`input_tokens` 1200 (1024 cached) and `output_tokens` 7.

Realism: Codex is a Responses client that streams every turn and reads
`response.completed.usage` to track how full the context window is (it
auto-compacts on it). Routed to any Chat provider (DeepSeek, Qwen, Gemini's
OpenAI layer, OpenAI in Chat mode) it sees an empty context on every turn.

Status: **new** (tracker search 2026-10-20 for `usage`, `include_usage`,
`response.completed`: nothing about this; #2052, #1923, #655, #456, #107 are
other bugs).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ResponsesStreamUsageAfterFinishChunk`
(`repro/internal/protocol/stream/zz_go2lean_repro_hunt2_test.go`, the
converter alone) and `TestGo2LeanRepro_GatewayResponsesStreamGetsUsage`
(`repro/internal/protocolserver/zz_go2lean_repro_hunt2_test.go`, through the
gateway; checks both the upstream request and the client's
`response.completed`).

Fix direction: emit the terminal events when the upstream stream ends (or on
the usage chunk), not at `finish_reason`; set `include_usage` on streaming
Responses -> Chat requests as `anthropic_view.go` does.

## General property

For every Chat chunk stream, the usage in the converter's
`response.completed` event equals the usage the stream carries, wherever the
usage chunk comes before the end of the stream (in particular after the
`finish_reason` chunk). And every streaming Responses -> Chat request asks
for usage (`stream_options.include_usage` is true).

Target functions: `(*chatToResponsesConverter).Next` (via
`NewChatToResponsesConverter`) for the first part,
`request.ConvertOpenAIResponsesToChat` together with the stage that sets
`stream` for the second.
