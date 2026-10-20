# Finding 45: Anthropic tool_choice "none" becomes auto towards Responses and Gemini (found by go2lean, blind specs; new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Anthropic client -> OpenAI provider in Responses mode, and Anthropic
client -> Google-style provider (request conversion; streaming or not).

Where (tingly-box `b788a3b`):

- Responses: `ConvertAnthropicBetaToolChoiceToResponses`
  (`internal/protocol/request/anthropic_beta_to_responses.go:204-233`) handles
  `auto`, `any` and `tool`, then falls through to "Default to auto"
  (`:229-232`); there is no `OfNone` case although Responses has `"none"`.
  It is live: `ConvertAnthropicBetaToResponsesRequest` calls it (`:71`),
  from the Anthropic -> Responses bridge
  (`internal/protocol/stage/anthropicbridge/responses.go:42`) and from
  `internal/protocol/transform/base.go:134`. The V1 twin
  `ConvertAnthropicV1ToolChoiceToResponses`
  (`internal/protocol/request/anthropic_v1_to_responses.go:363-392`, called
  from `transform/base.go:125`) does the same.
- Gemini: `convertAnthropicViewToGoogleRequest`
  (`internal/protocol/request/anthropic_view.go:773-776`) sets a tool config
  only for `auto`, `tool` and `any`; for `none` it stays nil, and Gemini
  applies its default, AUTO, although mode NONE exists.

Trigger: an Anthropic request with a custom tool and
`tool_choice: {"type":"none"}`.

Observed: Responses `"tool_choice":"auto"` (Beta and V1), Gemini
`toolConfig: null`, with the tool still declared in both.

Expected: Responses `"tool_choice":"none"`; Gemini
`functionCallingConfig.mode: NONE`.

Realism: medium. Anthropic SDK agents use `none` to force a text-only turn
(a final summary, an answer after the last tool result) while keeping the
tool list, and so the prompt cache, unchanged. The model may then call a
tool the client ruled out, which the agent loop executes or fails on. The
same mistake on other paths: `chat-request-params.md` (Anthropic -> Chat,
#10), `responses-anthropic-tool-choice-none.md` (#33),
`gemini-tool-choice-none.md` (#34); none of them covers these two
directions.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `tool_choice` with `none`, `FunctionCallingConfig`: nothing).
Reproduces on main `bc460f6`.

How it was found: by go2lean, from specs written blind (see the
[provenance note](README.md)):
`SpecConvertAnthropicV1ToResponsesRequestToolChoiceMeaning` and
`SpecConvertAnthropicToGoogleRequestToolChoiceMeaning` were refuted, `go
test` confirmed the counterexamples, and triage found no earlier finding
for these directions. The concrete counterexamples declared only server
tools; the repro tests use a realistic custom tool.

Evidence: `TestGo2LeanRepro_AnthropicToolChoiceNoneReachesResponses` and
`TestGo2LeanRepro_AnthropicToolChoiceNoneReachesGemini`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Anthropic request with tools, the target's tool choice keeps the
mode: `auto` -> auto, `any` -> required (Gemini ANY), `tool` -> that
function, `none` -> none (Gemini NONE).

Target functions: `request.ConvertAnthropicBetaToolChoiceToResponses`,
`request.ConvertAnthropicV1ToolChoiceToResponses` (via the request
converters), `request.ConvertAnthropicToGoogleRequest` and
`request.ConvertAnthropicBetaToGoogleRequest`.
