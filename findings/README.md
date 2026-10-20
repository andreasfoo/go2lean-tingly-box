# Findings

One page per finding. Status: **tracked** (an issue exists upstream),
**new** (no issue found; not reported yet), **not a bug** (explained).

| # | finding | path | status | evidence |
|---|---|---|---|---|
| 1 | [tool-call ids not truncated on one side](issue-1924-tool-call-ids.md) | Anthropic -> Chat request | tracked (#1924) | `poc/issue-1924/` |
| 2 | [system messages dropped](issue-1921-system-messages.md) | Anthropic -> Chat, Anthropic v1 -> Responses requests | tracked (#1921) | `poc/round2/` |
| 3 | [streamed tool calls all at index 0; max_tokens not "length"](issue-1923-stream-indexes.md) | Anthropic -> Chat stream | tracked (#1923) | `poc/round2/` |
| 4 | [reasoning_content empty without streaming](reasoning-content-nonstream.md) | Anthropic -> Chat non-stream | new | `poc/round3/` |
| 5 | [refusal lost when streaming](responses-chat-stream-refusal.md) | Responses -> Chat stream | new | direct test, below |
| 6 | [gateway environment credentials sent upstream](env-credentials-upstream.md) | upstream clients | new | `repro/internal/client/` |
| 7 | [cache reads counted twice without streaming](usage-nonstream-cached-input.md) | usage tracking, Chat passthrough | new | `repro/internal/protocolserver/` |
| 8 | [DeepSeek-style reasoning_content breaks thinking blocks](chat-anthropic-reasoning-null.md) | Chat -> Anthropic stream and non-stream | new | `repro/internal/protocol/{stream,nonstream}/` |
| 9 | [tool arguments lost when sent only in `.done`](responses-done-only-arguments.md) | Responses -> Chat and Responses -> Anthropic streams | new | `repro/internal/protocol/stream/` |
| 10 | [Anthropic -> Chat requests drop sampling, stop and tool_choice](chat-request-params.md) | Anthropic -> Chat request | new | `repro/internal/protocol/request/` |
| 11 | [Chat -> Anthropic stream opens a block while another is open](chat-anthropic-interleaved-blocks.md) | Chat -> Anthropic stream | found by go2lean, minor | `benchmark.md` |
| 12 | [recordings keep client credentials](recording-client-credentials.md) | recording | new, minor | code reading |
| 13 | [Responses clients get no usage when streaming from a Chat provider](responses-stream-trailing-usage.md) | Chat -> Responses stream, Responses -> Chat request | new | `repro/internal/protocol/stream/`, `repro/internal/protocolserver/` |
| 14 | [Chat tool call index taken from the Responses output index](responses-chat-tool-call-index.md) | Responses -> Chat stream | new | `repro/internal/protocol/stream/` |
| 15 | [Responses developer messages become user messages](responses-developer-message-as-user.md) | Responses -> Chat request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| 16 | [`model_context_window_exceeded` and `refusal` reported as normal completions](anthropic-abnormal-stop-reasons.md) | Anthropic -> Chat non-stream, Anthropic -> Responses stream and non-stream | new | `repro/internal/protocol/{stream,nonstream}/` |
| 17 | [Chat tool calls without an id stay without one](chat-tool-call-empty-id.md) | Chat -> Anthropic stream and non-stream, Chat -> Responses non-stream | new | `repro/internal/protocol/{stream,nonstream}/`, `repro/internal/protocolserver/` |
| 18 | [Chat -> Anthropic requests drop developer messages, parameterless tools, tool_choice modes](chat-anthropic-request-coverage.md) | Chat -> Anthropic request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| 19 | [Chat -> Responses requests drop max_completion_tokens, zero sampling values, parallel_tool_calls](chat-responses-request-params.md) | Chat -> Responses request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| 20 | [upstream errors not translated to the client's protocol](upstream-errors-not-translated.md) | error responses, all converted paths | new (related: #2052) | `repro/internal/protocolserver/` |
| 21 | [Anthropic tool schemas lose `$defs`, `additionalProperties` and other keywords](anthropic-tool-schema-keywords.md) | Anthropic -> Chat, Anthropic -> Responses request | new | `repro/internal/protocol/request/` |
| 22 | [structured output formats dropped](structured-output-dropped.md) | Chat <-> Responses <-> Anthropic requests, all six directions | new | `repro/internal/protocol/request/` |
| 23 | [an empty thinking block stays open across the answer](responses-anthropic-stream-open-thinking.md) | Responses -> Anthropic stream | new | `repro/internal/protocol/stream/` |
| 24 | [parameterless tool calls stream arguments `""`](parameterless-tool-stream-arguments.md) | Anthropic -> Chat and Anthropic -> Responses streams | new (related: #1840) | `repro/internal/protocol/stream/` |
| 25 | [PDF, document and file content dropped](document-content-dropped.md) | Anthropic -> Chat, Anthropic -> Responses, Chat -> Anthropic, Chat -> Responses requests | new | `repro/internal/protocol/request/` |
| 26 | [a refusal becomes `content: null`](responses-anthropic-nonstream-refusal.md) | Responses -> Anthropic non-stream | new (related: #912) | `repro/internal/protocol/nonstream/` |
| 27 | [one assistant turn becomes two assistant messages](responses-chat-split-assistant-turn.md) | Responses -> Chat request | new | `repro/internal/protocol/request/` |
| 28 | [reasoning summaries dropped without streaming](responses-anthropic-nonstream-reasoning-summary.md) | Responses -> Anthropic non-stream | new (related: #1863) | `repro/internal/protocol/nonstream/` |
| 29 | [images in tool output dropped](responses-anthropic-tool-output-image.md) | Responses -> Anthropic request | new | `repro/internal/protocol/request/` |
| 30 | [tool `strict` flag lost](chat-responses-tool-strict.md) | Chat -> Responses, Responses -> Chat requests | new | `repro/internal/protocol/request/` |
| 31 | [stream input usage depends on which fields message_delta repeats](usage-anthropic-delta-cache-creation.md) | usage accumulation, Anthropic provider streams | new, minor | `repro/internal/protocol/usage/` |
| 32 | [`ParseAnthropicResult` keeps only the last text block](sse-parse-anthropic-last-text.md) | test infrastructure (`sse` parsers) | new, minor | `repro/internal/protocol/sse/` |
| 33 | [tool_choice "none" becomes auto](responses-anthropic-tool-choice-none.md) | Responses -> Anthropic request | new (found by the go2lean M2 agent) | `repro/internal/protocol/request/` |
| 34 | [Gemini function calling mode NONE becomes auto](gemini-tool-choice-none.md) | Gemini -> Chat, Gemini -> Anthropic request (no gateway caller) | new, latent (found by the go2lean M2 agent) | `repro/internal/protocol/request/` |

Findings 6-11 came from reading code for the four spec directions
(usage, credentials, routing, config) and from the bug classes of similar
gateways (`../hunting-similar-tools.md`), not from a go2lean search; they
are marked so on their pages. Findings 13-20 (from
`responses-stream-trailing-usage` to `upstream-errors-not-translated`) were
also found by hand, in a second round of code reading over the converters
and error paths, and confirmed with `go test`; each page ends with the
general property the bug refutes, stated without the trigger values, and the
function it specifies. Findings 21-32 (from `anthropic-tool-schema-keywords`
to `sse-parse-anthropic-last-text`) were found by hand too, in a third round
of code reading over the request, stream and non-stream converters, the
usage accumulator and the result parsers, and confirmed with `go test`; their
pages follow the same form. Findings 33-34
(`responses-anthropic-tool-choice-none`, `gemini-tool-choice-none`) were not
found by hand: the go2lean M2 agent found them with its own spec
(MiniMax-M3); their repro tests were written afterwards.

Not bugs (the converters' intended repairs, recorded so they are not
reported): orphan tool calls get an "aborted" result, an unmatched
function_call_output becomes a user message (both in the Responses and
Anthropic round trips); a refusal is carried as text to Anthropic.
