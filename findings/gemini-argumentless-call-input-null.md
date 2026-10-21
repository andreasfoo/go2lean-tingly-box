# Finding 67: a Gemini function call without args becomes `input: null` / arguments `"null"` (new, not reported)

Severity: **medium** (see [the findings table](README.md)). Same class as
[finding 24](parameterless-tool-stream-arguments.md), on the Gemini paths.

When a Gemini `functionCall` has no `args`, the converters pass the nil map
through:

- Anthropic, streaming: `anthropicToolUseBlockStartWithInput`
  (`internal/protocol/stream/anthropic_wire_events.go:124-128`) turns a nil
  input into `"input": null` on purpose; used at
  `stream/google_to_any.go:310` (V1) and `:474` (Beta).
- Anthropic, non-stream: `nonstream/google_to_any.go:163` and `:257` set
  `"input": nil`.
- Chat: `json.Marshal` of the nil map gives `"null"`, at
  `stream/google_to_any.go:140` and `nonstream/google_to_any.go:44`.

Path: an Anthropic client (V1 or Beta) or a Chat client with a
Google-style provider (Gemini CLI, Antigravity, `api_style: google`),
streaming and non-stream (`protocolserver/protocol_dispatch.go:243-247`,
`:285-309`). How often Gemini omits `args` for a parameterless function
was not checked against the live API.

Trigger: `{"functionCall":{"id":"call_1","name":"get_time"}}` with
`finishReason: STOP`.

Observed: the Anthropic client's SDK accumulates `input: null`, and its
`ToParam` resends `"input": null`, which Anthropic rejects ("Input should
be a valid dictionary") when the conversation goes back to a Claude model;
the Chat client gets `arguments: "null"`, which client tool runners cannot
turn into keyword arguments.

Expected: `{}`, as for any other call without arguments.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md).

Repro: `TestGo2LeanRepro_GeminiArgumentlessCallInputIsObject`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`) and
`TestGo2LeanRepro_GeminiArgumentlessCallInputIsObjectNonStream`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_sdk_test.go`), V1,
Beta and Chat.
