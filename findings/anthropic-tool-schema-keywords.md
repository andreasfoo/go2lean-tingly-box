# Anthropic tool schemas lose `$defs`, `additionalProperties` and every other keyword (new, not reported)

Path: Anthropic client -> Chat provider and Anthropic client -> Responses
provider (request conversion; streaming or not).

Where (tingly-box `b788a3b`):

- Chat: `internal/protocol/request/anthropic_view.go:705`
  (`convertAnthropicToolViewsToOpenAI`) passes only
  `tool.InputSchema.Properties` and `tool.InputSchema.Required` to
  `convertAnthropicInputSchemaToOpenAIParameters`
  (`anthropic_v1_to_openai.go:30`), which builds the parameters from those
  two fields and `"type":"object"`.
- Responses: `ConvertAnthropicBetaToolsToResponses`
  (`anthropic_beta_to_responses.go:158`) and
  `ConvertAnthropicV1ToolsToResponses` (`anthropic_v1_to_responses.go:317`)
  build `{"type":"object","properties":…,"required":…}` the same way.

The Anthropic SDK keeps every other keyword of `input_schema` (`$defs`,
`$ref` targets, `additionalProperties`, `anyOf`, `description`, `$schema`, …)
in `InputSchema.ExtraFields` (`libs/anthropic-sdk-go/message.go:13181`); the
beta view carries them over (`viewAnthropicBetaToolInputSchema`,
`anthropic_view.go:202`), and no converter reads them.

Trigger: a tool whose `input_schema` has any top-level keyword other than
`type`, `properties` and `required`; for instance a nested model in `$defs`
referenced with `$ref`, or `additionalProperties: false`.

Observed: for the input schema
`{"type":"object","$defs":{"Attendee":{…}},"properties":{"attendees":{"type":"array","items":{"$ref":"#/$defs/Attendee"}}},"required":["attendees"],"additionalProperties":false}`,
the Chat function `parameters` and the Responses function `parameters` (beta
and v1) are both
`{"properties":{"attendees":{"items":{"$ref":"#/$defs/Attendee"},"type":"array"}},"required":["attendees"],"type":"object"}`:
`$defs` and `additionalProperties` are gone and the `$ref` points nowhere.

Expected: the function parameters are the input schema, every keyword with
its value.

Realism: high. Claude Code forwards MCP tools' schemas as they are, and
MCP servers built on pydantic (FastMCP, the Python MCP SDK) put nested models
in `$defs` and reference them with `$ref`; zod-generated schemas carry
`additionalProperties: false`. With `$defs` dropped the upstream either
rejects the schema (a `$ref` to a missing definition is an invalid schema)
or the model sees an item type it cannot resolve and invents its shape;
without `additionalProperties: false` a strict-mode provider cannot apply
strict mode to the tool.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_AnthropicToChatToolSchemaKeepsKeywords` and
`TestGo2LeanRepro_AnthropicToResponsesToolSchemaKeepsKeywords`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

Fix direction: start the parameters from `ExtraFields` and add `type`,
`properties` and `required` on top.

## General property

For every Anthropic request, the parameters of each converted function tool
are the tool's `input_schema` as a JSON object: every key the client sent
(not only `type`, `properties` and `required`) is present with the same
value.

Target functions: `request.ConvertAnthropicBetaToOpenAIRequest` (Chat),
`request.ConvertAnthropicBetaToolsToResponses` and
`request.ConvertAnthropicV1ToolsToResponses` (Responses).
