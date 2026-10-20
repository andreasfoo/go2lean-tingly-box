# Finding 58: Responses -> Chat requests drop parallel_tool_calls (found by go2lean, M2 agent specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Responses client -> Chat provider, request conversion.

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_responses_to_chat.go:17-77`
(`ConvertOpenAIResponsesToChat`) copies the model, instructions, input,
`max_output_tokens`, `temperature`, `top_p`, `reasoning.effort`, tools,
`tool_choice` and the prompt-cache fields, and never reads
`params.ParallelToolCalls`, although Chat has the same field. Live caller:
`internal/protocol/transform/base.go:105`, reached from the Responses
endpoint through `TransformOpenAIResponses`
(`internal/protocolserver/openai_responses.go:212`). The other caller,
`internal/protocol/stage/responsesbridge/chat.go:47`, is used only by
tests at `b788a3b`. The comment at
`internal/protocolserver/protocol_endpoint.go:52-55` accepts losing
*Responses-only* fields; `parallel_tool_calls` exists in both APIs.

Trigger: a Responses request with `parallel_tool_calls: false` and a
function tool, routed to a Chat provider (the default for unknown
OpenAI-compatible providers).

Observed (`{"model":"gpt-4.1","input":"What is the weather in Paris and Rome?","parallel_tool_calls":false,"tools":[{"type":"function","name":"get_weather",…}]}`):
`{"messages":[{"content":"What is the weather in Paris and Rome?","role":"user"}],"model":"gpt-4.1","max_tokens":4096,"tools":[…]}`,
with no `parallel_tool_calls`.

Expected: `"parallel_tool_calls": false` in the Chat request.

Realism: low. Agents that run tools one at a time set it; the provider
then falls back to its default, which allows parallel calls, and the client
can get several tool calls in one turn, which it asked not to get. The
request still succeeds; clients that handle several calls per turn anyway
see no error. This is the reverse direction of
`chat-responses-request-params.md` (#19, Chat -> Responses);
`anthropic-disable-parallel-tool-use.md` (#51) is the Anthropic analogue.

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by go2lean, from the M2 agent's own specs (see the
[provenance note](README.md)):
`SpecConvertOpenAIResponsesToChatParallelToolCalls` was refuted, `go test`
confirmed the counterexample, and triage found no earlier finding for this
direction. The repro test was written afterwards.

Evidence: `TestGo2LeanRepro_ResponsesToChatKeepsParallelToolCalls`
(`repro/internal/protocol/request/zz_go2lean_repro_m2_test.go`).

## General property

For every Responses request that sets `parallel_tool_calls`, the Chat
request has `parallel_tool_calls` with the same value.

Target function: `request.ConvertOpenAIResponsesToChat`.
