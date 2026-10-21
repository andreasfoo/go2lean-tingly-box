# Finding 64: Gemini -> Chat streams send parallel tool calls without an index, so the client merges them into one (new, not reported)

Severity: **high** (see [the findings table](README.md)).

`HandleGoogleToOpenAIStreamResponse`
(`internal/protocol/stream/google_to_any.go:129-160`) sends each Gemini
function call as a `tool_calls` delta whose tool-call object has `id`,
`type` and `function`, but no `"index"` key. A Chat client decodes the
missing index as 0 for every call, so all calls of one answer land on the
same tool call. Same class as [finding 3](issue-1923-stream-indexes.md) (the
Anthropic -> Chat stream), in a different function.

Path: a Chat client with a Google-style provider (Gemini CLI, Antigravity,
`api_style: google`), streaming: `protocolserver/protocol_dispatch.go:247`.
Gemini returns parallel function calls routinely. The non-stream path
(`nonstream/google_to_any.go`) is not affected.

Trigger: one Gemini answer with two parts,
`{"functionCall":{"id":"call_a","name":"get_weather","args":{"city":"Paris"}}}`
and `{"functionCall":{"id":"call_b","name":"get_time","args":{"tz":"Europe/Paris"}}}`.

Observed: openai-go's `ChatCompletionAccumulator` gives one tool call named
`get_weatherget_time` with arguments `{"city":"Paris"}{"tz":"Europe/Paris"}`,
which is not valid JSON.

Expected: two tool calls, `get_weather` and `get_time`, as go-genai's
`FunctionCalls()` gives for the same answer.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md); a sweep feeding every
X -> Chat stream to openai-go's accumulator found this one new (the other
paths showed #3 and #14).

Repro: `TestGo2LeanRepro_GeminiToChatParallelToolCalls`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`).
