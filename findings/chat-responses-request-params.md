# Finding 19: Chat -> Responses requests drop max_completion_tokens, zero sampling values and parallel_tool_calls (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Chat client -> OpenAI provider in Responses mode (request conversion;
streaming or not).

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_chat_to_openai_responses.go`,
`ConvertChatToOpenAIResponses`:

- `:87` reads only `params.MaxTokens`; `max_completion_tokens` is ignored
  (and the default is used instead when one is configured).
- `:94` and `:99` copy `temperature` and `top_p` only when `Value > 0`, so
  an explicit `0` is dropped and the upstream default (1) applies.
- `ParallelToolCalls` is never copied.

Same `> 0` pattern in `request/anthropic_v1_to_responses.go:52,57`
(`ConvertAnthropicV1ToResponsesRequest`); the beta conversion
(`anthropic_beta_to_responses.go:56-62`) uses `Valid()` and is right.

Trigger: a Chat request with `max_completion_tokens`, `temperature: 0`,
`top_p: 0` or `parallel_tool_calls: false`.

Observed: for `temperature 0, max_completion_tokens 50,
parallel_tool_calls false` the upstream gets
`{"input":[…],"model":"gpt-4.1","tool_choice":"auto","tools":[…]}`.

Expected: `"max_output_tokens":50`, `"temperature":0`,
`"parallel_tool_calls":false`.

Realism: `max_completion_tokens` is the current Chat field (OpenAI
deprecated `max_tokens`, and reasoning models reject it), so current SDKs
send it; the limit silently goes away. `temperature: 0` is the usual
"deterministic" setting of evals and extraction jobs. Agents that run tools
one at a time set `parallel_tool_calls: false`.

Status: **new** (tracker search 2026-10-20 for `max_completion_tokens`,
`temperature`, `parallel_tool_calls`: nothing about this).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ResponsesRequestKeepsTokensAndSampling`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt2_test.go`) and
`TestGo2LeanRepro_GatewayResponsesUpstreamGetsChatParams`
(`repro/internal/protocolserver/zz_go2lean_repro_hunt2_test.go`, the request
the upstream receives).

Fix direction: test `Valid()` instead of `> 0`; take
`max_completion_tokens` first, then `max_tokens`; copy `parallel_tool_calls`.

## General property

For every Chat request, each limit and sampling field that is set
(`max_completion_tokens`, else `max_tokens`; `temperature`; `top_p`;
`parallel_tool_calls`) appears in the Responses request with the same value,
including `0` and `false`; "set" means present, not positive.

Target functions: `request.ConvertChatToOpenAIResponses` (and, for
temperature and top_p, `request.ConvertAnthropicV1ToResponsesRequest`).
