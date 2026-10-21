# Finding 73: Gemini thought parts shown as part of the answer (new, not reported; latent)

Severity: **low** (see [the findings table](README.md)); latent.

The Gemini response converters never check `Part.Thought`: every text part
is appended to the answer text, thought summaries included
(`internal/protocol/stream/google_to_any.go:108`, `:284`, `:448`;
`nonstream/google_to_any.go:31`, `:151`, `:245`).

Path: every Gemini -> client converter (Chat, Anthropic V1 and Beta,
streaming and non-stream) with a Google-style provider. Latent: Gemini
returns thought text only when `includeThoughts` is set, and the genai
request path never sets a `ThinkingConfig` (none in
`request/any_to_google.go` or `request/anthropic_view.go`). Whether
Antigravity backends send thoughts unasked was not checked.

Trigger: a Gemini answer with
`{"text":"**Planning** The user wants a greeting.","thought":true}` and
`{"text":"Hello!"}`.

Observed: the Chat client's content and the Anthropic client's text are
"**Planning** The user wants a greeting.Hello!".

Expected: "Hello!", which is go-genai's `Text()` for the same answer (it
leaves thought parts out).

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md).

Repro: `TestGo2LeanRepro_GeminiThoughtPartsNotVisibleText`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`) and
`TestGo2LeanRepro_GeminiThoughtPartsNotVisibleTextNonStream`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_sdk_test.go`), Chat,
V1 and Beta.
