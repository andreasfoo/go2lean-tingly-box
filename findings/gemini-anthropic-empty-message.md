# Finding 82: Gemini -> Anthropic requests turn an image-only content into a message without content (new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

`convertGoogleContentToAnthropic`
(`internal/protocol/request/google_to_any.go:312-368`) converts text,
function calls and function responses and ignores `inlineData`; a content
whose parts are all images gives `NewUserMessage()` with no block
(`:361-367`), and `ConvertGoogleToAnthropicRequest` (`:255-309`) appends it
as it is. Anthropic rejects a message with empty content.

Path: Gemini request -> Anthropic request conversion. No gateway caller at
`b788a3b`: the Gemini -> X request converters are not used, so the bug is
latent, like [finding 56](gemini-anthropic-sampling-dropped.md).

Trigger: Gemini contents
`[{"role":"user","parts":[{"inlineData":{"mimeType":"image/png","data":"…"}}]},{"role":"user","parts":[{"text":"What is in this picture?"}]}]`.

Observed: `messages[0]` is `{"role":"user"}`, with no content.

Expected: the image as an image block (or no empty message).

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecConvertGoogleToAnthropicRequestIssue1251NoEmptyMessages` was refuted,
`go test` confirmed the counterexample, and the repro test was written
afterwards.

Repro: `TestGo2LeanRepro_GeminiToAnthropicNoEmptyMessage`
(`repro/internal/protocol/request/zz_go2lean_repro_issues1_test.go`).
