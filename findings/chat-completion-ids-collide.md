# Finding 74: Chat completion ids built from the Unix second, so answers in the same second share an id (new, not reported)

Severity: **low** (see [the findings table](README.md)).

The Chat id is `fmt.Sprintf("chatcmpl-%d", time.Now().Unix())` in
`internal/protocol/stream/google_to_any.go:52` (Gemini -> Chat stream),
`stream/anthropic_to_openai_converter.go:68` (Anthropic -> Chat stream) and
`nonstream/google_to_any.go:73` (Gemini -> Chat non-stream). Two answers
produced in the same second get the same id. (The Responses -> Chat stream
starts the same way but replaces it with the response id.)

Path: a Chat client with a Google-style or an Anthropic provider
(`protocolserver/protocol_dispatch.go:247` and the Anthropic -> Chat
stream). Answers to parallel requests often finish in the same second;
a client or log that keys on the completion id cannot tell them apart.

Trigger: three answers converted back to back.

Observed: two answers share `chatcmpl-1792561785`, on both paths.

Expected: a distinct id per answer.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md), while comparing the
Chat completions openai-go accumulates.

Repro: `TestGo2LeanRepro_ChatStreamIDsDistinct`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`), Gemini ->
Chat and Anthropic -> Chat.
