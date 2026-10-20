# Finding 3: Streamed tool calls at index 0; max_tokens not "length" (tracked: #1923)

Severity: **high** (see [the findings table](README.md)).

`anthropicToOpenAIConverter.processEvent`
(`internal/protocol/stream/anthropic_to_openai_converter.go`):

- two `tool_use` blocks in one message both stream as
  `tool_calls[0].index == 0` (`:152`, argument deltas at `:180`), so a
  Chat client merges parallel calls;
- a `message_delta` with stop reason `max_tokens` ends with
  `finish_reason` `"stop"`, not `"length"` (`:217-219`).

Specs: `...ParallelToolCallIndexes`, `...MaxTokensIsLength`. Confirmed;
PoCs in `poc/round2/`.
