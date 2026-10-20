# Finding 34: Gemini -> Chat and Gemini -> Anthropic: function calling mode NONE becomes auto (found by the go2lean M2 agent; new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

Path: Gemini request -> Chat provider and Gemini request -> Anthropic
provider (request conversion).

Where (tingly-box `b788a3b`), `internal/protocol/request/google_to_any.go`:

- `:242-246` (`ConvertGoogleToolChoiceToOpenAI`): mode `NONE` returns
  `tool_choice: "auto"`; the comment says "just don't include tools", but
  `ConvertGoogleToOpenAIRequest` includes them (`:54-64`).
- `:413-417` (`ConvertGoogleToolChoiceToAnthropic`): mode `NONE` returns
  `{"type":"auto"}`; the comment says "Anthropic doesn't have a direct
  'none' mode, so we don't pass tools", but Anthropic has the mode and
  `ConvertGoogleToAnthropicRequest` passes the tools (`:294-306`).

(The same two functions also map mode `ANY` without allowed names to auto,
`:238-241` and `:409-412`, where `ANY` means a call is required; not in the
test.)

Trigger: a Gemini `GenerateContentConfig` with function declarations and
`tool_config.function_calling_config.mode: NONE`.

Observed: the Chat request has the tool and `tool_choice: "auto"`; the
Anthropic request has the tool and `tool_choice: {"type":"auto"}`.

Expected: `"none"` for Chat and `{"type":"none"}` for Anthropic, with the
tools kept (or the tools removed, as the comments intend, but not both
kept and allowed).

Realism: low at `b788a3b`: the two request converters are exported but
have no caller outside their own tests (the gateway has no Gemini-protocol
client endpoint), so no request reaches them yet. A latent bug in
converters that a Gemini client endpoint would use as they are.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by the go2lean M2 agent with its own spec, MiniMax-M3,
not by hand; the repro test was written afterwards and confirmed with
`go test`.

Evidence: `TestGo2LeanRepro_GeminiToolChoiceNoneStaysNone`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Gemini function calling config, the converted `tool_choice` keeps
the mode: `AUTO` -> auto, `ANY` -> required (Chat) / `any` (Anthropic), or
the named function when exactly one is allowed, `NONE` -> `none`.

Target functions: `request.ConvertGoogleToolChoiceToOpenAI` and
`request.ConvertGoogleToolChoiceToAnthropic` (via
`request.ConvertGoogleToOpenAIRequest` and
`request.ConvertGoogleToAnthropicRequest`).
