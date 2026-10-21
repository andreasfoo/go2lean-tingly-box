# Finding 75: Anthropic -> Gemini requests drop `role:"system"` messages inside `messages` (new, not reported)

Severity: **high** (see [the findings table](README.md)).

`convertAnthropicViewMessageToGoogle`
(`internal/protocol/request/anthropic_view.go:784-866`) converts the user
and assistant roles only and returns nil for any other role, and
`convertAnthropicViewToGoogleRequest` (`:757-762`) skips a nil content. A
message with role `"system"` in `messages` is dropped without a trace. This
is the class of [finding 2](issue-1921-system-messages.md) (tracked as
#1921, Chat and Responses paths) on the Gemini path, which #1921 does not
name.

Path: an Anthropic V1 or Beta client with a Google-style provider
(`api_style: google`): `transform/base.go:218` and `:230`
(`convertToGoogle`), `protocolserver/protocol_dispatch.go:215`. Claude Code
2.1 puts its environment block (working directory, platform, git state) and
its token-budget notices in such messages on every request (#1921), so the
Gemini model never sees them.

Trigger: an Anthropic request with
`"messages":[{"role":"user","content":"hi"},{"role":"system","content":[{"type":"text","text":"# Environment\n - Primary working directory: /work/proj\n - Platform: linux"}]}]`
and no top-level `system`.

Observed: the Gemini request holds only the text `"hi"`; neither
`systemInstruction` nor `contents` carries the environment block (V1 and
Beta).

Expected: the text reaches Gemini, in `systemInstruction` or as text at the
same position, as #1921 asks for the Chat and Responses paths.

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecConvertAnthropicToGoogleRequestIssue1921KeepsSystemMessages` and its
Beta twin were refuted, `go test` confirmed the counterexamples, and the
repro test was written afterwards with Claude Code's request shape.

Repro: `TestGo2LeanRepro_AnthropicToGeminiKeepsInlineSystemMessages`
(`repro/internal/protocol/request/zz_go2lean_repro_issues1_test.go`), V1
and Beta.
