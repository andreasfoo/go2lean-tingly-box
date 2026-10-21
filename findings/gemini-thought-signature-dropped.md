# Finding 63: Gemini thoughtSignature dropped from function calls, so the next turn of a tool loop lacks it (new, not reported)

Severity: **high** (see [the findings table](README.md)).

Nothing in tingly-box reads or writes `Part.ThoughtSignature` (no
`ThoughtSignature` anywhere under `internal/`). The Gemini response
converters drop it: `internal/protocol/stream/google_to_any.go:129`, `:304`,
`:468` and `internal/protocol/nonstream/google_to_any.go:36`, `:158`,
`:252` copy only the function call's id, name and args. The request
converters that build the next Gemini request build `FunctionCall` parts
without it too: `request/any_to_google.go:133-148` (Chat client) and
`request/anthropic_view.go:843-858` (Anthropic client). `anthropic_view.go:856`
also skips thinking blocks, so the signature has no carrier at all through
an Anthropic or Chat client's history.

Path: an Anthropic client (Claude Code, V1 or Beta) or a Chat client with a
Google-style provider (Gemini CLI, Antigravity, `api_style: google`),
streaming (`protocolserver/protocol_dispatch.go:243-247`) and non-stream
(`:285-309`); the next request goes through `transform/base.go:218` and
`:242`. Every tool turn is affected: Gemini attaches a `thoughtSignature`
to the functionCall part of a thinking model's answer.

Trigger: a Gemini answer
`{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"Z2VtaW5pLXRob3VnaHQtc2lnbmF0dXJl"}`,
then the client's next request with the tool result.

Observed: after the client's SDK has accumulated the answer and resent it
(anthropic-sdk-go `Message.Accumulate` → `ToParam`, openai-go
`ChatCompletionAccumulator` → `ToParam`), the next Gemini request's
functionCall part has an empty `thoughtSignature`, for V1, Beta and Chat.

Expected: the signature Gemini sent, which is what a go-genai client
resending the model content would send. Gemini 3 rejects a request whose
function call lacks it (400 "Function call is missing a
thought_signature"), so an agent's tool loop stops at its second step (not
tested against the live API); on Gemini 2.5 the model loses its reasoning
context between steps.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md): the provider's answer
and tingly-box's converted output were fed to the SDK a client uses
(go-genai, anthropic-sdk-go, openai-go), and the two views compared.

Repro: `TestGo2LeanRepro_GeminiThoughtSignatureRoundTrips`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`), V1, Beta
and Chat, streaming.
