# Finding 24: Parameterless tool calls stream arguments `""` instead of `{}` (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Chat client -> Anthropic provider and Responses client -> Anthropic
provider, streaming.

Where (tingly-box `b788a3b`), `internal/protocol/stream/`:

- `anthropic_to_openai_converter.go:148`: a `tool_use` block starts the Chat
  tool call with `arguments: ""`; `:173` skips an `input_json_delta` whose
  `partial_json` is empty. A call without arguments therefore assembles to
  `""`.
- `anthropic_beta_to_openai_responses_converter.go:208` starts the
  `function_call` item with `arguments: ""`; `:298` sends the accumulated
  string (`""`) in `response.function_call_arguments.done`, and the same
  string is the item's `arguments` in `response.completed`.

The non-streaming converters give `"{}"` for the same call
(`nonstream/anthropic.go:36` marshals the `input` object;
`nonstream/openai_to_responses.go:164` defaults to `"{}"`).

Trigger: Claude calls a tool that takes no arguments. The Anthropic API
streams it as `content_block_start` with `input: {}` and one
`input_json_delta` with `partial_json: ""`.

Observed: the Chat tool call's assembled arguments are `""`; the Responses
`function_call_arguments.done` and the `response.completed` output carry
`arguments: ""`.

Expected: `"{}"`, as without streaming. Arguments are a JSON object
serialised as a string in both protocols.

Realism: parameterless tools are common (`get_time`, `list_files`,
`git_status`, MCP tools without input). A client that parses the arguments
(`json.loads`, `JSON.parse`, `serde_json::from_str`) fails on `""`; the
OpenAI SDKs' tool-runner helpers do exactly that.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20;
related, not the same: #1840, an open PR on `tool_use` input as an object).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ParameterlessToolChatStreamArgsAreJSON` and
`TestGo2LeanRepro_ParameterlessToolResponsesStreamArgsAreJSON`
(`repro/internal/protocol/stream/zz_go2lean_repro_hunt3_test.go`).

Fix direction: when a tool_use block stops with no argument text, emit
`{}` (or the block's start `input`, marshalled).

## General property

For every Anthropic event stream, the arguments of every tool call the
converter emits (the concatenation of a Chat tool call's argument deltas;
the Responses `function_call_arguments.done` and the completed item) are a
valid JSON object, equal as JSON to the tool_use input the upstream sent.

Target functions: `(*anthropicToOpenAIConverter).Next` (via
`NewAnthropicBetaToOpenAIChatConverter`) and
`(*anthropicBetaToResponsesConverter).Next` (via
`NewAnthropicBetaToOpenAIResponsesConverter`).
