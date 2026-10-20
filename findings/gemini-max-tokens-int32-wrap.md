# Finding 57: Anthropic and Chat -> Gemini: max_tokens above 2^31-1 wraps in maxOutputTokens (found by go2lean, M2 agent specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Anthropic (v1 and Beta) and Chat clients -> Google-style provider,
request conversion.

Where (tingly-box `b788a3b`):

- `internal/protocol/request/anthropic_view.go:742`
  (`convertAnthropicViewToGoogleRequest`):
  `config.MaxOutputTokens = int32(view.MaxTokens)`. Both
  `ConvertAnthropicToGoogleRequest` (`any_to_google.go:293-296`) and
  `ConvertAnthropicBetaToGoogleRequest` (`:311-314`) go through it.
- `internal/protocol/request/any_to_google.go:66-70`
  (`ConvertOpenAIToGoogleRequest`): `int32(req.MaxTokens.Value)` (`:67`) and
  `int32(defaultMaxTokens)` (`:69`).
- Live callers: `internal/protocol/transform/base.go:218,230,242`.
- The gateway's output limit does not repair it: `OutputLimitTransform` is a
  target transform, so it runs after the conversion
  (`internal/protocolserver/protocol_transform.go:250-261`), and its Google
  case (`internal/protocolserver/transform/transform_max_tokens.go:153-157`)
  compares the already wrapped int32. A wrapped small or negative value is
  below `MaxAllowed` and passes. On the Anthropic, Chat and Responses paths
  the same transform clamps the int64 field and works.

Trigger: a client sends a very large `max_tokens` to mean "no limit" (for
example JavaScript's `Number.MAX_SAFE_INTEGER`, or any value at or above
2^31), and the request is routed to a Google-style provider.

Observed (`{"model":"gemini-2.5-pro","max_tokens":N,"messages":[{"role":"user","content":"Write a long story."}]}`,
JSON-parsed as an Anthropic v1, Anthropic Beta and Chat request):

| `max_tokens` | `maxOutputTokens` (all three converters) |
|---|---|
| 8192 | 8192 |
| 4294967396 (2^32+100) | **100** |
| 9007199254740991 | **-1** |
| 3000000000 | **-1294967296** |

Expected: the value clamped (to the model limit or to `math.MaxInt32`)
before it is narrowed to int32, or the request rejected; never a wrapped
value.

Realism: low. The native Anthropic and OpenAI APIs reject such values, but
with a clear error; here a value that wraps to a small positive number
silently truncates the answer (2^32+100 becomes 100 tokens) and a negative
one makes Gemini reject the request with an error the client cannot
explain.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue). Not covered by `chat-gemini-request-coverage.md` (#47),
which is about `max_completion_tokens` being ignored.

How it was found: by go2lean, from the M2 agent's own specs (see the
[provenance note](README.md)): `SpecConvertAnthropicToGoogleRequestMaxTokens`
and `SpecConvertOpenAIToGoogleRequestMaxTokens` were refuted, `go test`
confirmed the counterexamples, and triage found no earlier finding for
them. The repro test was written afterwards.

Evidence: `TestGo2LeanRepro_GeminiMaxTokensDoesNotWrap`
(`repro/internal/protocol/request/zz_go2lean_repro_m2_test.go`).

## General property

For every Anthropic or Chat request converted to Gemini, `maxOutputTokens`
is positive and equals the requested limit whenever that limit fits in
int32; a larger limit is clamped, never wrapped.

Target functions: `request.ConvertAnthropicToGoogleRequest`,
`request.ConvertAnthropicBetaToGoogleRequest`,
`request.ConvertOpenAIToGoogleRequest`.
