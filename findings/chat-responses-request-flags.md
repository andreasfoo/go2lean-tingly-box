# Finding 53: Chat -> Responses requests drop user, safety_identifier, metadata, service_tier, store and top_logprobs (found by go2lean, blind specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Chat client -> Responses provider (an OpenAI provider in per-model
Responses mode, the Codex OAuth provider), request conversion.

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_chat_to_openai_responses.go:33-126`
(`ConvertChatToOpenAIResponses`) copies the model, the messages,
`max_tokens`, `temperature`, `top_p`, tools, `tool_choice`, the reasoning
effort and the prompt-cache fields (`:85-123`), and nothing else. Callers:
`internal/protocol/stage/openaibridge/responses.go:47`,
`internal/protocol/transform/base.go:144`.

Trigger: a Chat request with `"user":"user-42"`,
`"safety_identifier":"sid-42"`, `"metadata":{"session":"s1"}`,
`"service_tier":"flex"`, `"store":false`, `"logprobs":true,
"top_logprobs":3`.

Observed: none of the six reaches the Responses request.

Expected: all six carried (the Responses API has each field; `top_logprobs`
with the `message.output_text.logprobs` include).

Realism: low. `user` and `safety_identifier` are OpenAI's abuse
attribution, `metadata` tags requests for the client's dashboards, and
`service_tier` (`flex`, `priority`) sets the price and latency the client
chose. `store: false` matters most: Chat Completions does not store by
default, Responses stores unless `store` is false, so a client that opted
out of storage is stored by OpenAI on a plain OpenAI Responses provider
(the Codex path forces `store: false`,
`internal/protocol/ops/request_openai_codex.go:46`). Finding
`chat-responses-request-params.md` (#19) lists `max_completion_tokens`, zero
sampling values and `parallel_tool_calls` only.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)):
`SpecConvertChatToOpenAIResponsesKeepsRequestFlags` was refuted, `go test`
confirmed the counterexample, and triage found no earlier finding for
these fields. The repro test was written afterwards with a JSON-parsed
request.

Evidence: `TestGo2LeanRepro_ChatToResponsesKeepsRequestFlags`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Chat request, each of `user`, `safety_identifier`, `metadata`,
`service_tier`, `store` and `top_logprobs` that the request sets has the
same value in the Responses request.

Target function: `request.ConvertChatToOpenAIResponses`.
