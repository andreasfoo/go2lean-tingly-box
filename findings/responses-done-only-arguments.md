# Finding 9: Tool arguments lost when a Responses stream sends them only in `.done` (new, not reported)

Some Responses backends stream a function call without
`response.function_call_arguments.delta` events: the arguments arrive only
in `response.function_call_arguments.done` and `response.output_item.done`.
`gpt-5.3-codex-spark` on the ChatGPT Codex backend does this (raw streams
and per-model event counts in BerriAI/litellm#27144); tingly-box lists that
model for its Codex provider (#1269).

Both stream converters that read Responses streams drop those arguments:

- **Responses -> Chat** (`internal/protocol/stream/openai_responses_to_chat_converter.go`):
  `.done` and `output_item.done` store the arguments in the converter's
  state but emit no chunk, and `emitCompletedOutput` (the
  `response.completed` fallback) returns early because a tool call was
  already seen. The client gets the tool call with `arguments: ""` and
  `finish_reason: "tool_calls"`.
- **Responses -> Anthropic** (`openai_responses_to_anthropic_converter.go`):
  `.done` only stops the block, and `output_item.done` is ignored. The
  client gets `tool_use` with `input: {}` and `stop_reason: "tool_use"`.

So Claude Code (or a Chat client) routed to such a model runs the tool
with empty input on every call. The non-streaming converters read the
completed output and are not affected.

Fix direction: on `.done` (or `output_item.done`), when no delta was seen
for the item, emit the arguments as one delta before stopping the block.

How it was found: bug class 15 of `hunting-similar-tools.md`, checked by
feeding a `.done`-only stream to both converters.

Repro: `TestGo2LeanRepro_DoneOnlyArgumentsReachChatClient` and
`TestGo2LeanRepro_DoneOnlyArgumentsReachAnthropicClient`
(`repro/internal/protocol/stream/zz_go2lean_repro_resp_test.go`).

Tracker search (2026-10-20) for `spark`, `arguments.done`,
`function_call_arguments`: nothing about this.
