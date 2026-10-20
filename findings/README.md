# Findings

One page per finding. Status: **tracked** (an issue exists upstream),
**new** (no issue found; not reported yet), **not a bug** (explained).

| finding | path | status | evidence |
|---|---|---|---|
| [tool-call ids not truncated on one side](issue-1924-tool-call-ids.md) | Anthropic -> Chat request | tracked (#1924) | `poc/issue-1924/` |
| [system messages dropped](issue-1921-system-messages.md) | Anthropic -> Chat, Anthropic v1 -> Responses requests | tracked (#1921) | `poc/round2/` |
| [streamed tool calls all at index 0; max_tokens not "length"](issue-1923-stream-indexes.md) | Anthropic -> Chat stream | tracked (#1923) | `poc/round2/` |
| [reasoning_content empty without streaming](reasoning-content-nonstream.md) | Anthropic -> Chat non-stream | new | `poc/round3/` |
| [refusal lost when streaming](responses-chat-stream-refusal.md) | Responses -> Chat stream | new | direct test, below |
| [gateway environment credentials sent upstream](env-credentials-upstream.md) | upstream clients | new | `repro/internal/client/` |
| [cache reads counted twice without streaming](usage-nonstream-cached-input.md) | usage tracking, Chat passthrough | new | `repro/internal/protocolserver/` |
| [DeepSeek-style reasoning_content breaks thinking blocks](chat-anthropic-reasoning-null.md) | Chat -> Anthropic stream and non-stream | new | `repro/internal/protocol/{stream,nonstream}/` |
| [tool arguments lost when sent only in `.done`](responses-done-only-arguments.md) | Responses -> Chat and Responses -> Anthropic streams | new | `repro/internal/protocol/stream/` |
| [Anthropic -> Chat requests drop sampling, stop and tool_choice](chat-request-params.md) | Anthropic -> Chat request | new | `repro/internal/protocol/request/` |
| [Chat -> Anthropic stream opens a block while another is open](chat-anthropic-interleaved-blocks.md) | Chat -> Anthropic stream | found by go2lean, minor | `benchmark.md` |
| [recordings keep client credentials](recording-client-credentials.md) | recording | new, minor | code reading |
| [Responses clients get no usage when streaming from a Chat provider](responses-stream-trailing-usage.md) | Chat -> Responses stream, Responses -> Chat request | new | `repro/internal/protocol/stream/`, `repro/internal/protocolserver/` |
| [Chat tool call index taken from the Responses output index](responses-chat-tool-call-index.md) | Responses -> Chat stream | new | `repro/internal/protocol/stream/` |
| [Responses developer messages become user messages](responses-developer-message-as-user.md) | Responses -> Chat request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| [`model_context_window_exceeded` and `refusal` reported as normal completions](anthropic-abnormal-stop-reasons.md) | Anthropic -> Chat non-stream, Anthropic -> Responses stream and non-stream | new | `repro/internal/protocol/{stream,nonstream}/` |
| [Chat tool calls without an id stay without one](chat-tool-call-empty-id.md) | Chat -> Anthropic stream and non-stream, Chat -> Responses non-stream | new | `repro/internal/protocol/{stream,nonstream}/`, `repro/internal/protocolserver/` |
| [Chat -> Anthropic requests drop developer messages, parameterless tools, tool_choice modes](chat-anthropic-request-coverage.md) | Chat -> Anthropic request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| [Chat -> Responses requests drop max_completion_tokens, zero sampling values, parallel_tool_calls](chat-responses-request-params.md) | Chat -> Responses request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| [upstream errors not translated to the client's protocol](upstream-errors-not-translated.md) | error responses, all converted paths | new (related: #2052) | `repro/internal/protocolserver/` |

Findings 6-11 came from reading code for the four spec directions
(usage, credentials, routing, config) and from the bug classes of similar
gateways (`../hunting-similar-tools.md`), not from a go2lean search; they
are marked so on their pages. Findings 13-20 (from
`responses-stream-trailing-usage` to `upstream-errors-not-translated`) were
also found by hand, in a second round of code reading over the converters
and error paths, and confirmed with `go test`; each page ends with the
general property the bug refutes, stated without the trigger values, and the
function it specifies.

Not bugs (the converters' intended repairs, recorded so they are not
reported): orphan tool calls get an "aborted" result, an unmatched
function_call_output becomes a user message (both in the Responses and
Anthropic round trips); a refusal is carried as text to Anthropic.
