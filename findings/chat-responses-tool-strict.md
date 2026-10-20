# Tool `strict` flag lost between Chat and Responses (new, not reported)

Path: Chat client -> Responses provider and Responses client -> Chat
provider (request conversion; streaming or not).

Where (tingly-box `b788a3b`), `internal/protocol/request/`:

- `openai_chat_to_openai_responses.go:404-444`
  (`ConvertChatToolsToResponsesTools`) copies name, parameters and
  description, never `strict`.
- `openai_responses_to_chat.go:389-392`
  (`ConvertResponsesToolsToChatTools`): `if !param.IsOmitted(fn.Strict)`
  has an empty body ("strict mode is set via ExtraFields if needed");
  nothing is set.

Trigger: a function tool with `strict` set, `true` or `false`.

Observed: Chat `strict: false` reaches Responses as a tool without
`strict`; Responses `strict: true` reaches Chat as a tool without `strict`.

Expected: `strict` carried with its value in both directions; both APIs
have it on function tools.

Realism: the two APIs default an absent `strict` differently (Chat:
non-strict; Responses: strict where the schema allows it), so dropping it
does not just fall back to a neutral default. Responses -> Chat: a client
that asked for strict mode gets arguments no longer guaranteed to match the
schema. Chat -> Responses: a tool the client marked non-strict may be run
in strict mode, where optional parameters (not in `required`) are rejected
or forced. SDK helpers (`pydantic_function_tool`, zod helpers) set `strict`
explicitly.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ChatToResponsesKeepsStrict` and
`TestGo2LeanRepro_ResponsesToChatKeepsStrict`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

## General property

For every request, each converted function tool has `strict` set exactly
when the client's tool has it set, with the same value.

Target functions: `request.ConvertChatToolsToResponsesTools` and
`request.ConvertResponsesToolsToChatTools` (via
`request.ConvertChatToOpenAIResponses` and
`request.ConvertOpenAIResponsesToChat`).
