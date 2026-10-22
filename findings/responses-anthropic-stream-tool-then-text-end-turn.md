# Finding 83: Responses -> Anthropic streams end a tool call followed by text with `end_turn` (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

`responsesToAnthropicConverter.finalize`
(`internal/protocol/stream/openai_responses_to_anthropic_converter.go:586-591` at `bc460f6`) takes the stop reason from
`lastOutputItemType`, the type of the last output item seen: `tool_use`
only when the last item was a function, custom-tool or MCP call. A
response with a tool call followed by a message (text after the call, as
some models write a short note after calling) has `lastOutputItemType ==
"text"` (set at `:241` and `:253`), so the message ends with
`stop_reason: "end_turn"` although it contains a `tool_use` block.

Path: an Anthropic client with a Responses provider, streaming:
`stage/anthropicbridge/responses_stream.go:23`. An agent loop that runs
tools when `stop_reason` is `tool_use` (the Anthropic SDK's tool runner,
Claude Code) ends the turn instead and never runs the call.

Trigger: the Responses stream `output_item.added` (function_call
`get_weather`), `function_call_arguments.delta` / `.done`,
`output_item.done`, then a message item with the text `"Checking."`,
then `response.completed`.

Observed: a `tool_use` block, a text block, and `message_delta.stop_reason`
`"end_turn"`.

Expected: `"tool_use"`, whatever the order of the blocks (Anthropic sets it
when the message contains a tool call).

How it was found: while triaging go2lean's sequence specs
(`go2lean:sequence`, design §5.2 of go2lean). go2lean's M2 agent wrote
`SeqResponsesToAnthropicStopReason` for this converter blind (no findings,
no earlier specs); go2lean's search reported it violated, but on a
counterexample the model got wrong (the event payload goes through JSON
and the SDK union's `AsXxx`, which the model does not follow), and the
replay did not confirm it. The agent, asked to triage the result, ran the
converter on event sequences of its own in Go and found this one; the
repro test was written afterwards. So the spec stated the right property,
and go2lean's search did not find the bug.

Repro: `TestGo2LeanRepro_ResponsesToAnthropicToolThenTextIsToolUse`
(`repro/internal/protocol/stream/zz_go2lean_repro_seq_test.go`).
