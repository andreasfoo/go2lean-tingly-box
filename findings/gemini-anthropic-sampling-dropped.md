# Finding 56: Gemini -> Anthropic requests drop sampling parameters and stop sequences (found by go2lean, blind specs; new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

Path: Gemini request -> Anthropic request conversion. No gateway caller at
`b788a3b`: the Gemini -> X request converters are not used (no Gemini
client endpoint routes through them), so the bug is latent, like
`gemini-tool-choice-none.md` (#34).

Where (tingly-box `b788a3b`):
`internal/protocol/request/google_to_any.go:255-309`
(`ConvertGoogleToAnthropicRequest`) reads `MaxOutputTokens` (`:263-265`),
the contents, the tools and the tool config, and nothing else: no
`temperature`, `top_p`, `top_k` or `stop_sequences`. By reading,
`config.SystemInstruction` is ignored too (only contents of role "system"
become the system prompt, `:271-276`), here and in
`ConvertGoogleToOpenAIRequest` (`:15-68`), which does carry temperature
and top_p but not the stop sequences.

Trigger: a Gemini request with `temperature: 0.2`, `topP: 0.9`, `topK: 40`,
`stopSequences: ["END"]`.

Observed: `{"max_tokens":1024,"messages":[…],"model":"m"}`.

Expected: `temperature: 0.2`, `top_p: 0.9`, `top_k: 40`,
`stop_sequences: ["END"]`.

Realism: low at `b788a3b`, latent: it matters as soon as a Gemini client
endpoint is wired to these converters.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)):
`SpecConvertGoogleToAnthropicRequestKeepsSamplingParameters` was refuted,
`go test` confirmed the counterexample, and triage found no earlier
finding for it. The repro test was written afterwards.

Evidence: `TestGo2LeanRepro_GeminiToAnthropicKeepsSampling`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Gemini request, each of temperature, topP, topK and
stopSequences that the config sets has the same value in the Anthropic
request (`temperature`, `top_p`, `top_k`, `stop_sequences`).

Target function: `request.ConvertGoogleToAnthropicRequest`.
