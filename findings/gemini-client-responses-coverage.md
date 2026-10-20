# Finding 60: Anthropic and Chat -> Gemini responses drop thinking, refusals and cached prompt tokens (found by go2lean, M2 agent specs; new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

Path: Anthropic or Chat response -> Gemini response conversion, non-stream.
No gateway caller at `b788a3b`: no Gemini client endpoint routes through
`HandleAnthropicToGoogle` or `HandleOpenAIToGoogle` (nor through the stream
converters in `internal/protocol/stream/any_to_google.go`); the only
references outside these files are tests
(`internal/protocol/nonstream/nonstream_test.go`), and the gateway has no
Gemini-protocol client route (`generateContent` appears only in the
upstream clients, comments and the virtual upstream servers used by
benchmarks and tests). The bug is latent, like
`gemini-anthropic-sampling-dropped.md` (#56).

Where (tingly-box `b788a3b`), `internal/protocol/nonstream/any_to_google.go`:

- `HandleAnthropicToGoogle` (`:84-129`):
  - `:104-119` handles only `text` and `tool_use` blocks; a `thinking`
    block is dropped instead of becoming a `thought: true` part.
  - `:124-126` sets `promptTokenCount` from `input_tokens` only, which in
    Anthropic usage excludes the cache read and cache write tokens;
    `cachedContentTokenCount` is never set. Gemini's `promptTokenCount`
    includes the cached tokens.
  - `MapAnthropicFinishReasonToGoogle` (`:131-144`) maps `stop_sequence`
    (and `refusal`) to `OTHER`; it has a case for `content_filter`, which is
    not an Anthropic stop reason.
- `HandleOpenAIToGoogle` (`:12-66`) reads `Message.Content` only
  (`:35-37`); `Message.Refusal` is never read, so a refusal becomes an
  empty answer with `finishReason: "STOP"`.

Trigger: a Claude answer with a thinking block, a cached prompt and
`stop_reason: "stop_sequence"`; or an OpenAI completion with `content: null`
and a `refusal`.

Observed:
`{"candidates":[{"content":{"parts":[{"text":"Paris."}],"role":"model"},"finishReason":"OTHER"}],"usageMetadata":{"candidatesTokenCount":40,"promptTokenCount":20,"totalTokenCount":60}}`
for thinking + `Paris.`, usage `input_tokens` 20, `cache_read_input_tokens`
3000, `cache_creation_input_tokens` 500, `output_tokens` 40; and
`{"candidates":[{"content":{"role":"model"},"finishReason":"STOP"}],…}` for
the refusal.

Expected: a thought part with the thinking text, `finishReason: "STOP"`,
`promptTokenCount` 3520 with `cachedContentTokenCount` 3000; the refusal
text as a text part.

Realism: low at `b788a3b`, latent: it matters as soon as a Gemini client
endpoint is wired to these converters.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from the M2 agent's own specs (see the
[provenance note](README.md)): `SpecHandleAnthropicToGoogleConservation`,
`SpecHandleAnthropicToGoogleStopReason`, `SpecHandleAnthropicToGoogleUsage`
and `SpecHandleOpenAIToGoogleConservation` were refuted, `go test`
confirmed the counterexamples, and triage found no earlier finding for
them. The repro tests were written afterwards.

Evidence: `TestGo2LeanRepro_AnthropicToGeminiResponseKeepsThinkingAndCache`
and `TestGo2LeanRepro_ChatRefusalReachesGeminiClient`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_m2_test.go`).

## General property

For every Anthropic message converted to Gemini, text, thinking and
tool_use blocks reach the candidate's parts (thinking as thought parts),
normal terminations (`end_turn`, `stop_sequence`, `tool_use`) map to
`STOP`, and `promptTokenCount` is the whole prompt (uncached + cache read +
cache write) with the cache read as `cachedContentTokenCount`. For every
Chat completion converted to Gemini, the first choice's content, refusal
and tool calls reach the candidate's parts.

Target functions: `nonstream.HandleAnthropicToGoogle`,
`nonstream.HandleOpenAIToGoogle`.
