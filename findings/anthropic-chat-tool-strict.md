# Finding 52: tool `strict` flag lost between Anthropic and Chat / Responses (found by go2lean, blind specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Anthropic client -> Chat provider, Anthropic client -> Responses
provider, Chat client -> Anthropic provider (request conversion).

Where (tingly-box `b788a3b`):

- Anthropic -> Chat: `convertAnthropicToolViewsToOpenAI`
  (`internal/protocol/request/anthropic_view.go:690-714`) builds the
  `FunctionDefinitionParam` (`:707-711`) without `Strict`, although the view
  carries `tool.Strict` (`:193`).
- Anthropic -> Responses: `ConvertAnthropicBetaToolsToResponses`
  (`internal/protocol/request/anthropic_beta_to_responses.go:158-201`)
  builds the `FunctionToolParam` (`:188-193`) without `Strict`; the V1 twin
  (`anthropic_v1_to_responses.go:347`) too.
- Chat -> Anthropic: `ConvertOpenAIToAnthropicTools`
  (`internal/protocol/request/openai_to_anthropic.go:227-264`) builds the
  `BetaToolParam` (`:251-256`) without `Strict`.

Trigger: a tool with `"strict": true` (Anthropic strict tool use; a Chat
function with `strict: true`).

Observed: the flag is gone in all three directions.

Expected: `strict: true` carried (all three APIs have the field).

Realism: low. Strict tool use is set by SDK helpers (`pydantic_function_tool`,
zod helpers) and by Anthropic structured-output clients; without it the
arguments are no longer guaranteed to match the schema. Finding
`chat-responses-tool-strict.md` (#30) covers only Chat <-> Responses.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `strict`: nothing). Reproduces on main `bc460f6`.

How it was found: by go2lean, from specs written blind (see the
[provenance note](README.md)):
`SpecConvertAnthropicToOpenAIRequestKeepsToolDefinitions` and
`SpecConvertOpenAIToAnthropicRequestKeepsEveryFunctionTool` were refuted,
`go test` confirmed the counterexamples, and triage found no earlier
finding for these directions. The Anthropic -> Responses direction was
found by reading; the repro test was written afterwards.

Evidence: `TestGo2LeanRepro_AnthropicChatToolStrictCarried`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every request with function tools, each converted tool keeps the
`strict` flag of its source tool (present and equal, or absent when it was
absent).

Target functions: `request.ConvertAnthropicBetaToOpenAIRequest`,
`request.ConvertAnthropicBetaToResponsesRequest`,
`request.ConvertOpenAIToAnthropicRequest`.
