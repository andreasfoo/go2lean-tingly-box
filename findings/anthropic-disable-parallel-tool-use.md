# Finding 51: Anthropic disable_parallel_tool_use dropped towards Chat and Responses (found by go2lean, blind specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Anthropic client -> Chat provider, and Anthropic client -> Responses
provider (request conversion; streaming or not).

Where (tingly-box `b788a3b`): the normalized view keeps the flag
(`internal/protocol/request/anthropic_view.go:211-234`,
`viewAnthropicBetaToolChoice`), but

- Chat: `convertAnthropicViewToOpenAIRequest` (`anthropic_view.go:278-352`)
  and `convertAnthropicToolChoiceViewToOpenAI` (`:719-731`) never set
  `parallel_tool_calls`;
- Responses: `ConvertAnthropicBetaToResponsesRequest`
  (`internal/protocol/request/anthropic_beta_to_responses.go:12-102`) and
  its V1 twin never set `parallel_tool_calls` either.

Trigger: `tool_choice: {"type":"auto","disable_parallel_tool_use":true}`
with one custom tool.

Observed: no `parallel_tool_calls` in the Chat request nor in the
Responses request (both default to parallel calls allowed).

Expected: `parallel_tool_calls: false` on both.

Realism: low to medium. Agent loops that execute one tool at a time
(LangChain's Anthropic integration, custom loops with side-effecting
tools) set it; the model may then emit several calls in one turn. Finding
`chat-request-params.md` (#10) lists the other fields this conversion
drops, not this one.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `disable_parallel_tool_use`, `parallel_tool_calls`: nothing;
#1923 is stream indexes). Reproduces on main `bc460f6`.

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)):
`SpecConvertAnthropicToOpenAIRequestKeepsDisableParallelToolUse` was
refuted, `go test` confirmed the counterexample, and triage found no
earlier finding for it. The concrete counterexample declared only server
tools; the repro test uses a custom tool, and also covers the Responses
direction, found by reading.

Evidence: `TestGo2LeanRepro_DisableParallelToolUseCarried`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Anthropic request with tools whose tool_choice sets
`disable_parallel_tool_use: true`, the Chat and Responses requests set
`parallel_tool_calls: false`.

Target functions: `request.ConvertAnthropicBetaToOpenAIRequest`,
`request.ConvertAnthropicToOpenAIRequest`,
`request.ConvertAnthropicBetaToResponsesRequest`.
