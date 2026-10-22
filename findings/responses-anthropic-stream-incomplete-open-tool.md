# Finding 84: Responses -> Anthropic streams leave a tool_use block open when the response is cut short (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

`responsesToAnthropicConverter.emitStopEvents`
(`internal/protocol/stream/openai_responses_to_anthropic_converter.go:177-203` at `bc460f6`) closes the content blocks still open before
`message_delta`: thinking, refusal, summary and text, then
`r.state.pendingToolCalls` (`:191`). This converter never fills
`pendingToolCalls`: its tool calls live in `r.toolCalls` (`:346`) and are
closed only by `function_call_arguments.done` (or the custom-tool and MCP
`.done` events, `:364-394`). A response that ends while a call's
arguments are still streaming, `response.incomplete` with
`max_output_tokens` (`:405-415`), gives `message_delta` and
`message_stop` with the `tool_use` block never stopped.

Path: an Anthropic client with a Responses provider, streaming:
`stage/anthropicbridge/responses_stream.go:23`. A long tool argument (a
file written by a coding agent) running into `max_output_tokens` is the
common case. The Anthropic stream protocol requires every
`content_block_start` to be matched by a `content_block_stop`;
clients that parse a tool's input on `content_block_stop` never see this
block end.

Trigger: `output_item.added` (function_call `write_file`),
`function_call_arguments.delta` `{"path":"a.txt","content":"abc`, then
`response.incomplete` with `incomplete_details.reason:
"max_output_tokens"`.

Observed: `content_block_start` (index 0, `tool_use`),
`content_block_delta`, `message_delta` (`max_tokens`), `message_stop`;
no `content_block_stop` for index 0.

Expected: `content_block_stop` for index 0 before `message_delta`, as the
Chat -> Anthropic converter does for its pending calls
(`openai_to_anthropic_converter.go:439`).

How it was found: as [finding 83](responses-anthropic-stream-tool-then-text-end-turn.md):
while triaging go2lean's sequence specs for this converter
(`SeqResponsesToAnthropicStopReason`, `SeqResponsesToAnthropicContentConserved`,
violated in the model on counterexamples the replay did not confirm), the
agent ran the converter on event sequences of its own in Go. go2lean's
search did not find it.

Repro: `TestGo2LeanRepro_ResponsesToAnthropicIncompleteClosesToolBlock`
(`repro/internal/protocol/stream/zz_go2lean_repro_seq_test.go`).
