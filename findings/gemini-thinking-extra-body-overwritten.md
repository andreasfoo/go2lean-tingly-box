# Finding 40: Gemini thinking config replaces the request's `extra_body` (new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

Path: Chat request -> Gemini through its OpenAI-compatible endpoint
(`generativelanguage.googleapis.com`), with a thinking signal
(`reasoning_effort`, an effort derived from an Anthropic request, or a
`thinking` blob in the extra fields).

Where (tingly-box `b788a3b`):
`internal/protocol/ops/request_openai_gemini.go:133`
(`applyGeminiThinkingConfig`) sets
`extraFields["extra_body"] = {"google": googleConfig}`, replacing whatever
`extra_body` the request had. Called from `applyGeminiTransform` (`:57`),
which `ApplyProviderTransforms` runs for that host
(`internal/protocol/ops/request_openai_extensions.go:49-50`).

Trigger: a Chat request whose extra fields carry
`"extra_body": {"google": {…}}` (Google's documented way to pass Gemini
settings through the OpenAI-compatible API, for instance `cached_content`)
and a thinking signal such as `reasoning_effort: "low"`.

Observed: for `extra_body.google.cached_content: "cachedContents/abc123"`
and `reasoning_effort: "low"` on `gemini-2.5-flash`, the request's extra
fields become `{"extra_body":{"google":{"thinking_config":{"thinking_budget":4096}}}}`:
`cached_content` is gone (as would be any other `extra_body` key, or a
`thinking_config` with `include_thoughts` set by the client).

Expected: `extra_body.google` keeps every key it had and gains (or updates)
`thinking_config`.

Realism: low at `b788a3b`, latent. The gateway parses a Chat client's body
into `openai.ChatCompletionNewParams` (`internal/protocolserver/openai_chat.go:35`,
`protocol.OpenAIChatCompletionRequest.UnmarshalJSON`,
`internal/protocol/openai.go:41-56`), and the SDK's decoder does not keep
unknown fields: after parsing, `ExtraFields()` is empty and a client's
`extra_body` is already gone (checked with the vendored `libs/openai-go`).
No other code sets `extra_body` before this transform. So a client's
`cached_content` is lost at parsing, before this bug can drop it; the
transform drops it as soon as anything (a pass-through of unknown fields, a
rule's extra parameters) puts `extra_body` on the request.

Status: **new, latent** (tracker search 2026-10-20 over the full text of all
issues and PRs for `extra_body`: nothing). Reproduces on main `bc460f6`.

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_GeminiThinkingKeepsClientExtraBody`
(`repro/internal/protocol/ops/zz_go2lean_repro_hunt4_test.go`; it sets the
extra fields directly, since parsing drops them).

## General property

For every Chat request to Gemini, after the thinking transform every key
under `extra_body` and under `extra_body.google` of the request's extra
fields is still present with the same value, except `thinking_config`,
which the transform may set.

Target function: `ops.applyGeminiThinkingConfig` (via
`ops.ApplyProviderTransforms`).
