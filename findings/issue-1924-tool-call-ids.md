# Tool-call ids truncated on one side only (tracked: #1924)

`ConvertAnthropicBetaToOpenAIRequest` passes the `tool_use` id to the
assistant's `tool_calls[].id` unchanged
(`internal/protocol/request/anthropic_view.go` lines 94, 416 and 851 at
`5f83f04`) but truncates the result side
(`truncateToolCallID(block.ToolUseID)`, line 510). An Anthropic id longer
than the 40-character OpenAI limit therefore exceeds the limit on the
call side and no longer pairs with its tool message.

Counterexample: tool_use ids of 51, 50, 47 and 43 characters; the
converted request has an assistant tool call id longer than 40.

Spec: `SpecConvertAnthropicBetaToOpenAIRequestToolCallIDsBounded`
(`specs/internal/protocol/request/`). Found at trial 168, confirmed by go
test; PoC in `poc/issue-1924/`.
