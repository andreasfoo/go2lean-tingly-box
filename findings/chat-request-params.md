# Anthropic -> Chat requests drop sampling, stop and tool_choice (new, not reported)

`convertAnthropicViewToOpenAIRequest` (`internal/protocol/request/anthropic_view.go`),
which builds every Chat request for an Anthropic client, carries model,
max_tokens, messages, system, tools, cache hints and reasoning effort, and
nothing else:

- `temperature`, `top_p` and `stop_sequences` are not converted. Chat has
  all three (`temperature`, `top_p`, `stop`). The Anthropic -> Responses
  conversion (`anthropic_beta_to_responses.go:56-63`) does carry
  temperature and top_p, so the same request behaves differently by
  provider style.
- `tool_choice` becomes `"auto"` for everything but a named tool
  (`convertAnthropicToolChoiceViewToOpenAI`: "auto, any, and the default
  all map to auto"). `any` (the model must call a tool) should be
  `"required"`, which is what the Responses conversion sends; `none` (the
  model must not call a tool) should be `"none"` — as `"auto"` the model may
  call a tool the client explicitly ruled out.

Through the real gateway (`/tingly/anthropic/v1/messages` and
`/tingly/claude_code/v1/messages`, Chat provider) a request with
`"temperature":0,"top_p":0.5,"stop_sequences":["STOP"],"tool_choice":{"type":"none"}`
reaches the upstream as
`{"messages":[…],"model":"up","max_tokens":100,"tool_choice":"auto","tools":[…]}`.

How it was found: bug classes 26 and 29 of `hunting-similar-tools.md`
(tool_choice semantics and silently dropped request parameters, seen in
claude-code-proxy and litellm).

Repro: `TestGo2LeanRepro_ChatRequestKeepsSamplingAndStop` and
`TestGo2LeanRepro_ChatRequestKeepsToolChoiceMode`
(`repro/internal/protocol/request/zz_go2lean_repro_params_test.go`).

Tracker search (2026-10-20) for `temperature`, `stop_sequences`,
`tool_choice`: nothing about this.
