# Finding 17: Chat tool calls without an id reach Anthropic and Responses clients without one (new, not reported)

Path: Anthropic client -> Chat provider (streaming and non-streaming),
Responses client -> Chat provider (non-streaming).

Where (tingly-box `b788a3b`):

- `internal/protocol/stream/openai_to_anthropic_converter.go:257`
  (`processChunk`): `rewriteToolCallIDForAnthropic(toolCall.ID)`; an empty id
  stays empty.
- `internal/protocol/nonstream/openai_to_anthropic.go:96` (v1) and `:148`
  (`ConvertOpenAIChatToAnthropicBeta`): `NewToolUseBlock(toolCall.ID, …)`
  with the upstream id as is.
- `internal/protocol/nonstream/openai_to_responses.go:77`
  (`ConvertChatToResponsesWire`): `CallID: toolCall.ID`; the wire field is
  `call_id,omitempty` (`wire/openai_responses.go:147`), so the item has no
  `call_id` at all.

`ids.Call()` exists for exactly this and is used only by the Chat ->
Responses stream converter (`openai_chat_to_responses_converter.go:159`).

Trigger: a Chat completion (or stream) whose tool call has no `id` (or an
empty one).

Observed: the Anthropic client gets
`{"id":"","input":{"city":"Paris"},"name":"get_weather","type":"tool_use"}`
(streaming: `content_block_start` with `"id":""`); the Responses client gets
a `function_call` item without `call_id`.

Expected: a generated id (`toolu_…` / `call_…`), distinct per call, so that
the client's `tool_result` / `function_call_output` refers to its call.

Realism: Gemini's OpenAI compatibility layer returns tool calls without ids
(and some self-hosted OpenAI-compatible servers do too), and Gemini is a
common target behind tingly-box for Claude Code and Codex. The next request
then carries `tool_use_id: ""` / no `call_id`; with two parallel calls the
results cannot be told apart, and the Responses API (and Codex) require
`call_id` on `function_call_output`.

Status: **new** (tracker search 2026-10-20 for `tool call id`, `empty id`,
`call_id`: #1924 is ids too long for Anthropic, a different bug; nothing
about missing ids).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_StreamToolCallWithoutIDGetsID`
(`repro/internal/protocol/stream/zz_go2lean_repro_hunt2_test.go`),
`TestGo2LeanRepro_NonStreamToolCallWithoutIDGetsID`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_hunt2_test.go`, Responses
and Anthropic) and `TestGo2LeanRepro_GatewayToolCallWithoutIDGetsID`
(`repro/internal/protocolserver/zz_go2lean_repro_hunt2_test.go`, both clients
through the gateway).

Fix direction: generate an id with `ids.Call()` (or a `toolu_` id) when the
upstream id is empty, in each of the three places.

## General property

For every Chat completion or chunk stream, every tool call the converter
emits to the client (Anthropic `tool_use`, Responses `function_call`) has a
non-empty id, and the ids within one response are pairwise distinct, whatever
ids the upstream sent.

Target functions: `(*openAIToAnthropicConverter).Next` (via
`NewOpenAIChatToAnthropicBetaConverter`),
`nonstream.ConvertOpenAIChatToAnthropicBeta`,
`nonstream.ConvertOpenAIChatToAnthropicV1` and
`nonstream.ConvertChatToResponsesWire`.
