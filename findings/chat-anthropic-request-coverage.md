# Chat -> Anthropic requests drop developer messages, parameterless tools and tool_choice modes (new, not reported)

Path: Chat client -> Anthropic provider (request conversion; streaming or
not).

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_to_anthropic.go`,
`ConvertOpenAIToAnthropicRequest` (`:15`):

- The message switch has cases for system, user, assistant and tool
  messages and none for `OfDeveloper`: developer messages are dropped.
- `ConvertOpenAIToAnthropicTools` (`:237`) skips a function tool whose
  `parameters` is absent (`fn.Parameters == nil`); in Chat `parameters` is
  optional (a tool without arguments).
- `ConvertOpenAIToAnthropicToolChoice` (`:266-300`) maps `"auto"`,
  `"required"`, `"none"` and custom choices all to `{"type":"auto"}`; only a
  named function becomes `{"type":"tool"}`.
- `parallel_tool_calls`, `temperature`, `top_p` and `stop` are never read
  (Anthropic has `disable_parallel_tool_use`, `temperature`, `top_p`,
  `stop_sequences`).

Trigger: a Chat request with a developer message, a function tool without
`parameters`, `tool_choice` `"required"` or `"none"`, or any of
`parallel_tool_calls: false`, `temperature`, `top_p`, `stop`.

Observed: for messages `[developer "Answer in French.", user "What time is
it?"]`, tools `get_time` (no parameters) and `get_weather`, and
`tool_choice: "required"`, the upstream gets
`{"max_tokens":4096,"messages":[{"content":[{"text":"What time is it?","type":"text"}],"role":"user"}],"model":"claude-sonnet-4-5","tool_choice":{"type":"auto"},"tools":[{…"name":"get_weather"}]}`.

Expected: the developer text in `system`; both tools (`input_schema`
`{"type":"object"}` for the parameterless one); `tool_choice`
`{"type":"any"}` for `"required"` and `{"type":"none"}` for `"none"`;
`disable_parallel_tool_use: true` for `parallel_tool_calls: false`;
`temperature`, `top_p` and `stop` (as `stop_sequences`) carried.

Realism: the OpenAI SDKs and many agents send their system prompt as a
developer message (OpenAI's guidance for o-series and gpt-5 models), so the
whole prompt is lost on a Claude provider. Parameterless tools (`get_time`,
`list_files`) are common; `"required"` is how agent loops force a tool
call, and `"none"` how they forbid one (as `"auto"` the model may call a
tool the client ruled out). This is the reverse direction of
`chat-request-params.md`.

Status: **new** (tracker search 2026-10-20 for `developer`, `tool_choice`,
`parameters`, `temperature`: nothing about this direction; #1921 is system
messages on the Anthropic -> Chat path).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_AnthropicRequestKeepsDeveloperAndTools` and
`TestGo2LeanRepro_AnthropicRequestKeepsSamplingAndStop`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt2_test.go`), and
`TestGo2LeanRepro_GatewayAnthropicUpstreamGetsChatParams`
(`repro/internal/protocolserver/zz_go2lean_repro_hunt2_test.go`, the request
the upstream receives).

## General property

For every Chat request: the text of every system and developer message is in
the Anthropic `system`, every function tool is in `tools`, the `tool_choice`
mode is kept (`auto` -> `auto`, `required` -> `any`, `none` -> `none`, a
named function -> `tool`), `parallel_tool_calls: false` sets
`disable_parallel_tool_use`, and `temperature`, `top_p` and `stop` are
carried with their values whenever they are set.

Target function: `request.ConvertOpenAIToAnthropicRequest`.
