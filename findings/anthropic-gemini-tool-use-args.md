# Finding 44: Anthropic -> Gemini: tool_use arguments dropped from every function call (found by go2lean, blind specs; new, not reported)

Severity: **high** (see [the findings table](README.md)).

Path: Anthropic client (V1 and Beta) -> Google-style provider (`api_style:
google`: the Gemini CLI and Antigravity OAuth providers, which tingly-box
points at `cloudcode-pa.googleapis.com`,
`internal/server/module/oauth/handler.go:1238-1244`, and any user-configured
Google-style provider), streaming or not.

Where (tingly-box `b788a3b`):
`internal/protocol/request/anthropic_view.go:843-855`
(`convertAnthropicViewMessageToGoogle`, assistant branch): the tool_use
input becomes the function call's `Args` only
`if inputBytes, ok := block.OfToolUse.Input.([]byte)` (`:846-848`). A
request the gateway parses from JSON
(`internal/protocolserver/anthropic_message.go:72`, `:86`, `json.Unmarshal`
into the SDK params) holds `Input` as a `map[string]interface{}`, for V1
and Beta alike, so the assertion always fails and `Args` stays nil (`:853`).
Both `ConvertAnthropicToGoogleRequest` and
`ConvertAnthropicBetaToGoogleRequest`
(`internal/protocol/request/any_to_google.go:293-296`, `:311-314`) go
through it; the gateway calls them from `convertToGoogle`
(`internal/protocol/transform/base.go:216-238`), which `dispatchGoogle`
(`internal/protocolserver/protocol_dispatch.go:215`) runs for Google-style
providers; Anthropic clients are allowed there
(`internal/protocolserver/attempt_plan.go:107-111`).

Trigger: any Anthropic request whose history holds an assistant tool_use,
i.e. every request of a tool loop after the first:
`{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Paris"}}`.

Observed: `{"functionCall":{"id":"toolu_1","name":"get_weather"}}`, no
`args`, in both V1 and Beta.

Expected: `{"functionCall":{"id":"toolu_1","name":"get_weather","args":{"city":"Paris"}}}`.

Realism: high. Claude Code, or any Anthropic agent, routed to a Gemini CLI
or Antigravity provider resends its whole history on every step of a tool
loop; Gemini then sees each of its earlier calls without arguments, and
the tool results no longer answer a meaningful call.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from specs written blind (see the
[provenance note](README.md)): `SpecConvertAnthropicToGoogleRequestKeepsToolUseArgs`
and `SpecConvertAnthropicBetaToGoogleRequestKeepsToolUseArgs` were refuted,
`go test` confirmed the counterexamples, and triage found no earlier
finding for it. The repro test was written afterwards with a JSON-parsed
request.

Evidence: `TestGo2LeanRepro_AnthropicToGeminiKeepsToolUseArgs`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

Fix direction: marshal whatever `Input` holds (`json.Marshal`, then
unmarshal into the map), as the Chat and Responses converters do.

## General property

For every Anthropic request (V1 or Beta) and every assistant tool_use block
whose input is a JSON object, the Google request holds a functionCall with
the same id and name whose args equal that input.

Target functions: `request.ConvertAnthropicToGoogleRequest` and
`request.ConvertAnthropicBetaToGoogleRequest` (via
`request.convertAnthropicViewMessageToGoogle`).
