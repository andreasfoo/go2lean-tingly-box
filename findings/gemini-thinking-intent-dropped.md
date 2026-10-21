# Finding 79: the client's thinking request never reaches a native Gemini provider (new, not reported)

Severity: **low** (see [the findings table](README.md)).

`convertAnthropicViewToGoogleRequest`
(`internal/protocol/request/anthropic_view.go:737-779`) and
`ConvertOpenAIToGoogleRequest` (`request/any_to_google.go:60-200`) never
set `GenerateContentConfig.ThinkingConfig`: an Anthropic
`thinking.budget_tokens` and a Chat `reasoning_effort` are dropped. Nothing
later sets it either: `ops.ApplyThinkingEffort`
(`ops/request_thinking.go:30-65`) has no case for the Gemini request, so a
rule's `thinking_effort` does nothing there too (by reading); only the
probe (`probe/helper.go:558`) sets a thinking config. #1897 and #1917
carried the client's thinking intent between Anthropic, Chat and Responses;
`ops/request_openai_gemini.go:87-120` does it for Gemini's OpenAI-compatible
endpoint, not for the native API.

Path: an Anthropic V1 or Beta client or a Chat client with a Google-style
provider: `transform/base.go:218`, `:230`, `:242`. The budget or effort is
ignored and `includeThoughts` is never set, so the client never gets
thinking back; models that do not think by default (Flash-Lite, 2.0) do
not think at all.

Trigger: `{"max_tokens":16000,"thinking":{"type":"enabled","budget_tokens":10240},"messages":[{"role":"user","content":"Prove that sqrt(2) is irrational."}]}`,
or a Chat request with `"reasoning_effort":"high"`.

Observed: the Gemini `generationConfig` has no `thinkingConfig` (Anthropic
V1, Beta and Chat).

Expected: a `thinkingConfig` from the budget or effort (`thinkingBudget`
or `thinkingLevel`, `includeThoughts: true`), on the same ladder
`request_openai_gemini.go` uses.

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecConvertAnthropicToGoogleRequestIssue1897ThinkingCarried`, its Beta
twin and `SpecConvertOpenAIToGoogleRequestIssue1897EffortCarried` were
refuted, `go test` confirmed the counterexamples, and the repro test was
written afterwards.

Repro: `TestGo2LeanRepro_ThinkingIntentReachesGemini`
(`repro/internal/protocol/request/zz_go2lean_repro_issues1_test.go`).
