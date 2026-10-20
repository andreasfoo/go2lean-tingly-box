# Finding 37: Codex without streaming: function calls dropped from the assembled response (new, not reported)

Severity: **high** (see [the findings table](README.md)).

Path: Responses client (passthrough) and Chat client -> ChatGPT Codex
provider, not streaming. The Codex backend only streams, so tingly-box
streams and assembles the response itself.

Where (tingly-box `b788a3b`):

- `internal/client/codex_client.go:100-110` (`(*CodexClient).ResponsesNew`)
  streams and assembles the events with `assembler.NewResponsesAssembler()`
  (`:588`, in `parseResponsesStream`).
- `internal/protocol/assembler/openai_responses_assembler.go:390-398`
  (`ensureResponseHasAccumulatedOutput`): when the terminal
  `response.completed` has an empty `output`, the output is filled by
  `syntheticOutputItems` (`:409-434`), which builds one assistant message
  from the streamed text and refusal; function calls (and every other item:
  reasoning, …) are not added. A stream without text or refusal keeps the
  empty output (`:394-396`). `response.output_item.done` items are not
  collected (`:132-149` keeps only image results).
- Callers: `nonstreamOpenAIResponses`
  (`internal/protocolserver/protocol_passthrough.go:151`, Responses client)
  and `nonstreamResponsesToChat` (`internal/protocolserver/protocol_cross.go:98`,
  Chat client), both through `forwarding.ForwardOpenAIResponses`.
- The Stage pipeline used for Anthropic clients assembles the same backend
  correctly: `completeFromStream` (`internal/protocol/stage/upstream/openai.go:210-257`)
  collects the `response.output_item.done` items and uses them when the
  final output is empty (`:240-252`).

Trigger: a non-streaming request with tools to a Codex provider; the
backend's stream delivers a message and a function call as output items and
ends with `response.completed` whose `output` is `[]` (as Codex does).

Observed: the assembled response has output `[message]`: the function call
is gone. The Chat client gets the text with `finish_reason` "stop" and no
`tool_calls`; with no text before the call, the output is empty.

Expected: output `[message, function_call]`: every item the stream
delivered, as `completeFromStream` assembles it.

Realism: high. Every tool call of a non-streaming client (agent frameworks,
scripts, eval harnesses with `stream: false`) routed to a Codex provider is
lost; the model's turn ends with a preamble and no action.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `codex` with `non-stream`, `function_call`, `assembler`:
nothing; #982, merged, introduced the non-stream Codex path through the
assembler). Reproduces on main `bc460f6`.

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_CodexNonStreamKeepsFunctionCall`
(`repro/internal/protocol/assembler/zz_go2lean_repro_hunt4_test.go`).

Fix direction: collect `response.output_item.done` items in the assembler
and use them when the terminal output is empty, as `completeFromStream` does.

## General property

For every Responses event stream that ends with a terminal response, the
assembled response's output is the terminal output when that is non-empty,
and otherwise the items of the `response.output_item.done` events in
output-index order: every function call the stream delivered is in the
output.

Target function: `(*assembler.ResponsesAssembler).Finish` (after
`Accumulate`; used by `client.(*CodexClient).ResponsesNew`).
