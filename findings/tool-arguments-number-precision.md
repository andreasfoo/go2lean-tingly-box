# Finding 66: tool-argument integers above 2^53 rounded on Chat and Responses <-> Anthropic conversions (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Tool-call arguments are decoded into `interface{}` / `map[string]any`
without `UseNumber`, so every number becomes a `float64` and integers above
2^53 are rounded when the input is marshalled again:

- requests: `internal/protocol/request/openai_to_anthropic.go:97` (Chat
  client) and `request/openai_responses_to_anthropic.go:316` (Responses
  client);
- non-stream responses: `nonstream/openai_to_anthropic.go:93` (Chat -> Anthropic
  V1), `:145` (Beta) and `:193` (Responses -> Anthropic).

The Chat -> Anthropic stream forwards `partial_json` verbatim
(`stream/openai_to_anthropic_converter.go:271-276`), so the same answer is
exact when streamed and rounded when not.

Path: a Chat or Responses client with an Anthropic provider (requests:
`protocolserver/stage_openai_route.go:80`, `:82`,
`stage/openaibridge/bridge.go:59`, `stage/responsesbridge/bridge.go:53`);
an Anthropic client with a Chat or Responses provider, non-stream
(`stage/anthropicbridge/bridge.go:119`, `stage/anthropicbridge/responses.go:76`).
Realistic triggers are 64-bit ids in tool arguments: Discord and Twitter
snowflakes, database keys. The tool then acts on a different object,
silently.

Trigger: tool-call arguments `{"id":9007199254740993}`.

Observed: the upstream Anthropic request and the Anthropic client's
response (decoded with `UseNumber`) both carry `{"id":9007199254740992}`.

Expected: `9007199254740993`, the value the client or the provider sent.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md): the arguments the
provider's or client's SDK holds, compared with what reaches the other
side.

Repro: `TestGo2LeanRepro_ToolArgsNumbersExactInAnthropicRequest`
(`repro/internal/protocol/request/zz_go2lean_repro_sdk_test.go`), Chat and
Responses clients; `TestGo2LeanRepro_ChatToAnthropicToolArgsExactNonStream`
(V1 and Beta) and `TestGo2LeanRepro_ResponsesToAnthropicToolArgsExactNonStream`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_sdk_test.go`).
