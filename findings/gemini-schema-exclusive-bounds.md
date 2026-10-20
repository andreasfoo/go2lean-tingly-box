# Finding 39: Gemini tool schema filter: exclusive bounds overwrite minimum/maximum at random, nested schemas not visited (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: any client -> Gemini through its OpenAI-compatible endpoint
(`generativelanguage.googleapis.com`) or through Poe (Chat request, tool
schemas; `ApplyProviderTransforms`,
`internal/protocol/ops/request_openai_extensions.go:49-53`).

Where (tingly-box `b788a3b`), `internal/protocol/ops/request_openai_gemini.go`:

- `:43-46` and `:256-258` (`filterGeminiSchema`): `exclusiveMinimum` and
  `exclusiveMaximum` are written to `minimum` and `maximum`, inside the loop
  over the schema map. When a schema has both `minimum` and
  `exclusiveMinimum`, whichever Go's map iteration visits last wins, so the
  output differs from run to run. With the boolean form
  (`"exclusiveMinimum": true`, JSON Schema draft 4 / OpenAPI 3.0) the
  output can be `"minimum": true`, which is not a number.
- `:267-296`: only `properties`, `items` and `anyOf` are visited
  recursively; `additionalProperties`, `$defs` / `definitions`, `allOf`,
  `oneOf`, `not`, `prefixItems` and the rest are copied as they are
  (`default`, `:294-295`), so `exclusiveMinimum` / `exclusiveMaximum` stay
  in those subschemas although the filter exists to remove them (`:227-228`).

Trigger: a tool whose parameters have, for one number, both `minimum` and
`exclusiveMinimum` (or the maximum pair); or an exclusive bound in a
subschema under `additionalProperties`, `$defs`, `allOf`, `oneOf`, ….

Observed: for `{"type":"integer","minimum":1,"exclusiveMinimum":true}`, 500
runs give two outputs, `{"minimum":1,"type":"integer"}` and
`{"minimum":true,"type":"integer"}`. For a schema with
`additionalProperties: {"type":"number","exclusiveMinimum":0}` and
`$defs.Item.properties.qty.exclusiveMinimum: 0`, both `exclusiveMinimum`
keys are still in the output.

Expected: one output for one input (deterministic), with numeric
`minimum` / `maximum`, and no `exclusiveMinimum` / `exclusiveMaximum` at
any depth.

Realism: medium. Pydantic (`gt=0`, `lt=…`) and zod (`.positive()`) emit
exclusive bounds, and pydantic puts nested models in `$defs`, so MCP tools
built on them reach the unvisited branches (a Chat client's tool
parameters arrive as sent; on the Anthropic -> Chat path, finding 21
already drops a top-level `$defs`, but subschemas inside `properties` are
kept); the boolean form comes from OpenAPI-3.0-based tool generators. The filter exists because Gemini does
not take these keywords (`:227-228`); where they survive, or where
`minimum` becomes `true` (not a valid schema), the request can be rejected,
and in the two-keys case only on some runs, which is hard to diagnose.

Status: **new** (not reported; round 4 had no access to the tracker's
search, so it is not yet checked for an existing issue).

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_GeminiSchemaExclusiveBoundDeterministic` and
`TestGo2LeanRepro_GeminiSchemaNestedExclusiveBoundRemoved`
(`repro/internal/protocol/ops/zz_go2lean_repro_hunt4_test.go`).

## General property

For every JSON schema s, `filterGeminiSchema(s)` is a function of s (equal
inputs give equal outputs, independently of map iteration order), its
output has no `exclusiveMinimum` / `exclusiveMaximum` key in any subschema
(under `properties`, `items`, `prefixItems`, `additionalProperties`,
`$defs` / `definitions`, `anyOf`, `allOf`, `oneOf`, `not`), and every
`minimum` / `maximum` in the output is a number.

Target function: `ops.filterGeminiSchema` (via
`ops.applyGeminiToolSchemaFilter`, from `ops.ApplyProviderTransforms`).
