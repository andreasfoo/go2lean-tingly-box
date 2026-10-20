# Finding 49: Gemini thoughtsTokenCount never counted as output (found by go2lean, blind specs; new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Anthropic and Chat clients -> Google-style provider with a thinking
model (Gemini 2.5 and later), response usage, non-stream and stream; and
the gateway's own usage record for those requests.

Where (tingly-box `b788a3b`): `UsageMetadata.ThoughtsTokenCount` is read
nowhere in `internal/`. Output is taken from `CandidatesTokenCount` alone:

- `internal/protocol/nonstream/google_to_any.go:270-281`
  (`HandleGoogleToAnthropicBeta`), `:176-186` (`HandleGoogleToAnthropic`),
  `:92-97` (`HandleGoogleToOpenAI`, where `completion_tokens` excludes the
  thoughts but `total_tokens` includes them, so the three do not add up);
- `internal/protocolserver/protocol_dispatch.go:272-280` (the usage the
  gateway records for a non-stream Google request);
- by reading, the stream converters:
  `internal/protocol/stream/google_to_any.go:192`, `:338`, `:364`, `:502`,
  `:545`.

Trigger: a Gemini response with
`usageMetadata: {promptTokenCount: 1000, cachedContentTokenCount: 900,
candidatesTokenCount: 50, thoughtsTokenCount: 400, totalTokenCount: 1450}`.

Observed: Anthropic `output_tokens: 50`; Chat `completion_tokens: 50`,
`total_tokens: 1450`.

Expected: 450 output tokens (Gemini bills thoughts as output; OpenAI and
Anthropic count reasoning in the output tokens).

Realism: medium. Every call to a Gemini thinking model through a
Google-style provider; thoughts are often most of the billed output, so
clients and the usage dashboard undercount output, sometimes by a factor
of ten. The same response also double counts cached input
(`usage-nonstream-cached-input.md`, #7, which names the Gemini paths "by
reading only"); this finding is the output half.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `thoughtsTokenCount`, `thought tokens`: nothing; #1181, merged,
fixed the cached-token half on Google -> Anthropic responses, not thoughts).
Reproduces on main `bc460f6`.

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)): `SpecHandleGoogleToAnthropicBetaUsage` was
refuted, `go test` confirmed the counterexample, and triage split it into
the known #7 (input) and this new output bug. The repro test was written
afterwards with a JSON-parsed response.

Evidence: `TestGo2LeanRepro_GeminiThoughtsTokensCountedAsOutput`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_blind_test.go`).

## General property

For every Gemini response, the client's output tokens equal
`candidatesTokenCount + thoughtsTokenCount`, and for Chat clients
`prompt_tokens + completion_tokens = total_tokens`.

Target functions: `nonstream.HandleGoogleToAnthropicBeta`,
`nonstream.HandleGoogleToAnthropic`, `nonstream.HandleGoogleToOpenAI`.
