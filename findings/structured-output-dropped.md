# Finding 22: Structured output formats dropped in every request direction (new, not reported)

Path: every converted request path: Chat <-> Responses, Chat <->
Anthropic, Responses <-> Anthropic (request conversion; streaming or not).

Where (tingly-box `b788a3b`), `internal/protocol/request/`:

- Chat -> Responses: `ConvertChatToOpenAIResponses`
  (`openai_chat_to_openai_responses.go:33`) never reads `response_format`.
- Responses -> Chat: `ConvertOpenAIResponsesToChat`
  (`openai_responses_to_chat.go:17`) never reads `text.format`.
- Chat -> Anthropic: `ConvertOpenAIToAnthropicRequest`
  (`openai_to_anthropic.go:15`) never reads `response_format`; the only
  `output_config` field it sets is `effort` (`:336`).
- Responses -> Anthropic: `ConvertOpenAIResponsesToAnthropicBetaRequest`
  (`openai_responses_to_anthropic.go:21`) never reads `text.format`.
- Anthropic -> Chat: `viewAnthropicBetaRequest` (`anthropic_view.go:262-264`)
  copies only `output_config.effort` into the view, and the view converter
  reads only the effort (`:367`).
- Anthropic -> Responses: `ConvertAnthropicBetaToResponsesRequest`
  (`anthropic_beta_to_responses.go:12`) never reads `output_config.format`.

Trigger: a request with a JSON-schema output format: Chat
`response_format: {"type":"json_schema","json_schema":{…}}`, Responses
`text.format: {"type":"json_schema",…}`, Anthropic
`output_config.format: {"type":"json_schema","schema":…}`.

Observed: the converted requests carry no format at all, for instance Chat
-> Responses gives
`{"input":[{"content":"give json","role":"user","type":"message"}],"model":"m","tool_choice":"auto"}`
and Responses -> Anthropic
`{"max_tokens":1024,"messages":[…],"model":"m"}`.

Expected: the format in the target protocol's field with the same schema:
Chat `response_format` (`json_schema.name`, `schema`, `strict`) <->
Responses `text.format` (`name`, `schema`, `strict`) <-> Anthropic
`output_config.format` (`schema`); `json_object` where the target has it.

Realism: high. Structured output is how the OpenAI SDKs' `parse()` helpers
(`chat.completions.parse`, `responses.parse` with pydantic or zod models),
instructor and LangChain's `with_structured_output` work, and Anthropic's
structured outputs use `output_config.format`. Through any converted path
the model answers in free text and the client's parse fails, or a
schema-shaped reply comes back only by luck.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ChatResponseFormatCarried`,
`TestGo2LeanRepro_ResponsesTextFormatCarried` and
`TestGo2LeanRepro_AnthropicOutputFormatCarried`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`), one per
client protocol, each checking both providers.

## General property

For every request that sets a structured output format, the converted
request sets the target protocol's output format field, with the same
format type and the same JSON schema.

Target functions: `request.ConvertChatToOpenAIResponses`,
`request.ConvertOpenAIResponsesToChat`,
`request.ConvertOpenAIToAnthropicRequest`,
`request.ConvertOpenAIResponsesToAnthropicBetaRequest`,
`request.ConvertAnthropicBetaToOpenAIRequest`,
`request.ConvertAnthropicBetaToResponsesRequest`.
