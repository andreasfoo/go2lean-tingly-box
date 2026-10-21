# Finding 65: Chat and Responses -> Anthropic requests send a tool_use without `input` when the arguments are empty, invalid or `null` (new, not reported)

Severity: **high** (see [the findings table](README.md)); medium once
[finding 24](parameterless-tool-stream-arguments.md), its common trigger,
is gone.

`ConvertOpenAIToAnthropicRequest`
(`internal/protocol/request/openai_to_anthropic.go:94-99`) leaves
`argsInput` nil when a replayed tool call's `arguments` is `""`, invalid
JSON or `"null"`; `convertResponsesFunctionCallToAnthropicBeta`
(`request/openai_responses_to_anthropic.go:313-320`) does the same for `""`
and `"null"` (invalid JSON there becomes `{}`). The block is then built
with `NewBetaToolUseBlock(id, nil, name)`, and the SDK's `Input any
'omitzero'` drops the key: the request carries a `tool_use` block with no
`input`.

Path: a Chat client or a Responses client (Codex) with an Anthropic
provider: `protocolserver/stage_openai_route.go:80` and `:82`,
`stage/openaibridge/bridge.go:59`, `stage/responsesbridge/bridge.go:53`.
The common chain: Claude calls a parameterless tool, tingly-box streams it
to the Chat or Responses client as `arguments: ""`
([finding 24](parameterless-tool-stream-arguments.md)), the client replays
that call in its history, and from then on every request of the
conversation is rejected.

Trigger: a Chat history with
`{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_message","arguments":""}}]}`
and its tool result (or the Responses equivalent, a `function_call` item
with `"arguments":""`); also `"null"`, and `{"city":` on the Chat path.

Observed: the marshalled Anthropic request has
`{"type":"tool_use","id":"call_1","name":"get_message"}` with no `input`;
Anthropic answers 400 "input: Field required".

Expected: an `input` object (`{}` for an empty call).

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md): the client's history
as its SDK (openai-go) sends it, converted, and checked against what the
Anthropic API accepts.

Repro: `TestGo2LeanRepro_ToolUseInputPresentInAnthropicRequest`
(`repro/internal/protocol/request/zz_go2lean_repro_sdk_test.go`), Chat and
Responses clients, `""`, invalid JSON and `"null"`.

Further trigger, response direction: the Chat -> Anthropic non-stream
converters (`internal/protocol/nonstream/openai_to_anthropic.go:91-96`,
`:143-148`) unmarshal the upstream's `arguments` into a nil map; `""` and
invalid JSON become `{}`, but `"null"` (what tool parsers that serialise a
missing value send for a parameterless call) leaves the map nil, so the
Anthropic client gets `"input": null`, V1 and Beta
(`stage/anthropicbridge/bridge.go:119`). Replayed, that turn hits this
finding's request path. Found by go2lean from the issue-derived specs
(`SpecConvertOpenAIChatToAnthropicBetaIssue1840ToolInputIsObject`). Repro:
`TestGo2LeanRepro_ChatToAnthropicNullArgumentsInputIsObject`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_issues1_test.go`).
