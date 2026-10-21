# Finding 80: Gemini -> Chat responses drop cached prompt tokens (new, not reported)

Severity: **low** (see [the findings table](README.md)).

`HandleGoogleToOpenAI`
(`internal/protocol/nonstream/google_to_any.go:92-97`) builds the Chat
`usage` from `promptTokenCount`, `candidatesTokenCount` and
`totalTokenCount` only; `cachedContentTokenCount` is never read, so there
is no `prompt_tokens_details.cached_tokens`. #1181 mapped the cached tokens
for Anthropic clients; the Chat map was left out. This is not
[finding 7](usage-nonstream-cached-input.md) (the gateway's own usage
record) or [finding 49](gemini-thoughts-tokens-dropped.md) (thought
tokens).

Path: a Chat client with a Google-style provider, non-stream:
`protocolserver/protocol_dispatch.go:309`. The client cannot see cache
hits on Gemini's implicit and explicit caching.

Trigger: a Gemini answer with
`"usageMetadata":{"promptTokenCount":1000,"cachedContentTokenCount":900,"candidatesTokenCount":50,"totalTokenCount":1050}`.

Observed: `"usage":{"prompt_tokens":1000,"completion_tokens":50,"total_tokens":1050}`.

Expected: also `"prompt_tokens_details":{"cached_tokens":900}`.

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecHandleGoogleToOpenAIIssue1181CachedTokens` was refuted, `go test`
confirmed the counterexample, and the repro test was written afterwards
with a JSON-parsed Gemini answer.

Repro: `TestGo2LeanRepro_GeminiToChatKeepsCachedTokens`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_issues1_test.go`).
