# Finding 61: Anthropic -> Gemini requests drop temperature, top_p, top_k and stop_sequences (new, not reported)

Severity: **medium** (see [the findings table](README.md)). The Gemini
counterpart of [finding 10](chat-request-params.md).

`convertAnthropicViewToGoogleRequest`
(`internal/protocol/request/anthropic_view.go:737-778`), which builds every
Gemini request for an Anthropic client (V1 and Beta), sets
`maxOutputTokens`, the system instruction, contents, tools and tool config,
and nothing else. The normalised `anthropicRequestView` (`:30-43`) has no
fields for `temperature`, `top_p`, `top_k` or `stop_sequences`, so they are
lost before the conversion starts; no later step on the Gemini path sets
them. Gemini supports all four (`generationConfig.temperature`, `topP`,
`topK`, `stopSequences`).

Path: an Anthropic client (Claude Code, V1 or Beta) with a Google-style
provider (Gemini CLI or Antigravity OAuth, or `api_style: google`),
streaming or not: `protocolserver/anthropic_message.go` →
`dispatchGoogle` (`protocol_dispatch.go:215`) → `transform/base.go:216-238`
→ `request.ConvertAnthropicToGoogleRequest` /
`ConvertAnthropicBetaToGoogleRequest` (`request/any_to_google.go:293-314`).

Trigger:
`{"model":"gemini-2.5-pro","max_tokens":256,"temperature":0.2,"top_p":0.9,"top_k":40,"stop_sequences":["END"],"messages":[{"role":"user","content":"Say hi"}]}`
gives `generationConfig` `{"maxOutputTokens":256}` for V1 and Beta;
expected `{"maxOutputTokens":256,"temperature":0.2,"topP":0.9,"topK":40,"stopSequences":["END"]}`.
Without them the model writes past the client's stop sequence, and an
agent's temperature 0 is lost.

How it was found: the go2lean M2 agent's spec
`SpecConvertAnthropicToGoogleRequestSamplingConserved`, searched again with
the current model (run `specs5`, 133 agent specs, 50 confirmed); a triage
of the 50 found this one new, the rest known or unreachable.

Repro: `TestGo2LeanRepro_GeminiRequestKeepsSampling`
(`repro/internal/protocol/request/zz_go2lean_repro_m2b_test.go`).
