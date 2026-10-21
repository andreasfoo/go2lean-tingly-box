# Finding 76: Chat and Responses -> Anthropic requests send tool-call ids Anthropic rejects (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

`ConvertOpenAIToAnthropicRequest`
(`internal/protocol/request/openai_to_anthropic.go:100`, tool_result
`:119`) and `ConvertOpenAIResponsesToAnthropicBetaRequest`
(`request/openai_responses_to_anthropic.go:326`, tool_result `:348`) copy
the client's tool-call id into `tool_use.id` and `tool_result.tool_use_id`
unchanged. The Anthropic API accepts only `^[a-zA-Z0-9_-]+$` there, and no
transform rewrites these ids (PR #1401, draft, does it for
`server_tool_use` only).

Path: a Chat client or a Responses client (Codex) with an Anthropic
provider: `protocolserver/stage_openai_route.go:80` and `:82`,
`stage/openaibridge/bridge.go:59`, `stage/responsesbridge/bridge.go:53`.
The ids come from the provider that produced the history: Kimi names its
tool calls `functions.<name>:<n>`, and tingly-box passes upstream ids
through to the client. Once a conversation started on such a provider is
routed to Claude (a rule or model switch, a failover), every request of the
conversation is rejected.

Trigger: a Chat history with
`{"role":"assistant","content":"","tool_calls":[{"id":"functions.read_file:0","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]}`
and `{"role":"tool","tool_call_id":"functions.read_file:0","content":"package main"}`
(or the Responses equivalent, `function_call` and `function_call_output`
items with `"call_id":"functions.read_file:0"`).

Observed: `tool_use.id` and `tool_result.tool_use_id` are
`functions.read_file:0`; Anthropic answers 400 "tool_use.id: String should
match pattern '^[a-zA-Z0-9_-]+$'".

Expected: both ids rewritten to the allowed charset with the same
deterministic function, so they still pair.

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecConvertOpenAIToAnthropicRequestIssue1401ToolIDsAnthropicPattern` and
`SpecConvertOpenAIResponsesToAnthropicBetaRequestIssue1401ToolIDsAnthropicPattern`
were refuted, `go test` confirmed the counterexamples, and the repro test
was written afterwards with Kimi-style ids.

Repro: `TestGo2LeanRepro_ForeignToolIDsSanitizedForAnthropic`
(`repro/internal/protocol/request/zz_go2lean_repro_issues1_test.go`), Chat
and Responses clients.
