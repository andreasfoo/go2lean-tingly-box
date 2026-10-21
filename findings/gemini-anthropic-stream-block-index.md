# Finding 62: Gemini -> Anthropic streams number the first tool_use block 0, after a text block 0 (new, not reported)

Severity: **high** (see [the findings table](README.md)).

`HandleGoogleToAnthropicStreamResponse` and
`HandleGoogleToAnthropicBetaStreamResponse`
(`internal/protocol/stream/google_to_any.go:244-305`, `:408-469`) count text
and tool blocks separately: the first text block gets index 0
(`textBlockIndex = 0`), and tool blocks count up from -1
(`toolBlockIndex++`), so the first tool_use block also gets index 0. The text
block is never closed before the tool block starts, and two
`content_block_stop` events for index 0 follow.

Path: an Anthropic client (Claude Code, V1 or Beta) with a Google-style
provider (Gemini CLI, Antigravity, `api_style: google`), streaming:
`protocolserver/protocol_dispatch.go:243-245`. Any turn in which Gemini
writes text before calling a tool (the usual shape of an agent turn) is
affected.

What the client sees, by its own SDK: anthropic-sdk-go's
`Message.Accumulate` rejects the stream ("received event of type
content_block_start for content block at index 0, expected index 1"), and
the accumulated message holds only the text: the tool call is lost. The
same stream also ends with `stop_reason: "end_turn"` instead of `tool_use`,
so a client that does accept it will not run the tool.

How it was found: the SDK issue trackers (anthropic-sdk-go's accumulator
checks block indexes; cc-switch had the same class) suggested using the
clients' SDKs as the oracle for "a client through tingly-box sees what the
provider said"; this is the first property checked that way.

Repro: `TestGo2LeanRepro_GeminiToAnthropicStreamAccumulates`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`), V1 and
Beta, with the Anthropic SDK's `Message.Accumulate` as the client.
