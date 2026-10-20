# Finding 43: `StreamTokenCounter.Reset` keeps the upstream usage (new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

Path: the streaming token counter (`token.StreamTokenCounter`), used by the
Chat -> Anthropic stream converter
(`internal/protocol/stream/openai_to_anthropic_converter.go:105`); `Reset`
has no production caller at `b788a3b` (only
`internal/protocol/stream/verification_test.go:122`).

Where (tingly-box `b788a3b`): `internal/protocol/token/stream_counter.go:225-231`
(`Reset`) zeroes `inputTokens`, `outputTokens` and the pending text, but not
the upstream fields (`upstreamInputTokens`, `upstreamOutputTokens`,
`upstreamCacheTokens`, `upstreamCacheWriteTokens`, `upstreamReasoning`,
`:36-40`). `GetCounts` (`:174-189`) prefers the upstream values when they
are non-zero, and `GetUpstreamDetails` (`:194-198`) returns them.

Trigger: a counter that consumed a chunk with upstream usage, then `Reset`.

Observed: after a chunk with prompt 100 and completion 50 tokens and
`Reset`, `GetCounts` returns (100, 50).

Expected: (0, 0): a reset counter reports nothing, like a new one.

Realism: low, latent. A caller that reuses one counter across streams
(the doc comment's usage pattern with a reset in between) reports the
previous stream's usage for a stream whose upstream sends none.

Status: **new** (not reported; round 4 had no access to the tracker's
search, so it is not yet checked for an existing issue).

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_StreamCounterResetClearsUpstream`
(`repro/internal/protocol/token/zz_go2lean_repro_hunt4_test.go`).

## General property

For every sequence of calls on a `StreamTokenCounter` followed by `Reset`,
the counter's observable state (`GetCounts`, `GetUpstreamDetails`,
`TotalTokens`) equals that of a newly created counter.

Target function: `(*token.StreamTokenCounter).Reset` (read through
`GetCounts` and `GetUpstreamDetails`).
