# Findings

One page per finding. Status: **tracked** (an issue exists upstream),
**new** (no issue found; not reported yet), **not a bug** (explained).

Severity: **critical** — a security problem (credentials reach a third
party); **high** — a common client and provider path breaks or silently
loses what the user asked for (requests rejected, tools or instructions
lost, protocol violations a client trips over); **medium** — wrong results
on a less common path, or a loss with a visible symptom; **low** — rare
triggers, latent code, test infrastructure. The table is sorted by
severity; `#` is the finding's fixed number (the order it was recorded in).

| # | severity | finding | path | status | evidence |
|---|---|---|---|---|---|
| 6 | critical | [gateway environment credentials sent upstream](env-credentials-upstream.md) | upstream clients | new | `repro/internal/client/` |
| 1 | high | [tool-call ids not truncated on one side](issue-1924-tool-call-ids.md) | Anthropic -> Chat request | tracked (#1924) | `poc/issue-1924/` |
| 2 | high | [system messages dropped](issue-1921-system-messages.md) | Anthropic -> Chat, Anthropic v1 -> Responses requests | tracked (#1921) | `poc/round2/` |
| 3 | high | [streamed tool calls all at index 0; max_tokens not "length"](issue-1923-stream-indexes.md) | Anthropic -> Chat stream | tracked (#1923) | `poc/round2/` |
| 8 | high | [DeepSeek-style reasoning_content breaks thinking blocks](chat-anthropic-reasoning-null.md) | Chat -> Anthropic stream and non-stream | new | `repro/internal/protocol/{stream,nonstream}/` |
| 9 | high | [tool arguments lost when sent only in `.done`](responses-done-only-arguments.md) | Responses -> Chat and Responses -> Anthropic streams | new | `repro/internal/protocol/stream/` |
| 13 | high | [Responses clients get no usage when streaming from a Chat provider](responses-stream-trailing-usage.md) | Chat -> Responses stream, Responses -> Chat request | new | `repro/internal/protocol/stream/`, `repro/internal/protocolserver/` |
| 14 | high | [Chat tool call index taken from the Responses output index](responses-chat-tool-call-index.md) | Responses -> Chat stream | new | `repro/internal/protocol/stream/` |
| 21 | high | [Anthropic tool schemas lose `$defs`, `additionalProperties` and other keywords](anthropic-tool-schema-keywords.md) | Anthropic -> Chat, Anthropic -> Responses request | new | `repro/internal/protocol/request/` |
| 22 | high | [structured output formats dropped](structured-output-dropped.md) | Chat <-> Responses <-> Anthropic requests, all six directions | new | `repro/internal/protocol/request/` |
| 23 | high | [an empty thinking block stays open across the answer](responses-anthropic-stream-open-thinking.md) | Responses -> Anthropic stream | new | `repro/internal/protocol/stream/` |
| 25 | high | [PDF, document and file content dropped](document-content-dropped.md) | Anthropic -> Chat, Anthropic -> Responses, Chat -> Anthropic, Chat -> Responses requests | new | `repro/internal/protocol/request/` |
| 35 | high | [Anthropic Beta metadata transform panics on a user_id without account](anthropic-metadata-empty-account-panic.md) | Anthropic Beta client -> `api.anthropic.com` provider with an API key | new | `repro/internal/protocol/ops/` |
| 36 | high | [budget-thinking models: thinking stripped from the tool turn while thinking stays on](anthropic-budget-thinking-stripped.md) | Anthropic client -> Anthropic / Claude Code provider (vendor transform), V1 and Beta | new | `repro/internal/protocol/ops/` |
| 37 | high | [Codex without streaming drops function calls](codex-nonstream-function-calls-dropped.md) | Chat and Responses clients -> Codex provider, non-stream | new | `repro/internal/protocol/assembler/` |
| 4 | medium | [reasoning_content empty without streaming](reasoning-content-nonstream.md) | Anthropic -> Chat non-stream | new | `poc/round3/` |
| 5 | medium | [refusal lost when streaming](responses-chat-stream-refusal.md) | Responses -> Chat stream | new | direct test, below |
| 7 | medium | [cache reads counted twice without streaming](usage-nonstream-cached-input.md) | usage tracking, Chat passthrough | new | `repro/internal/protocolserver/` |
| 10 | medium | [Anthropic -> Chat requests drop sampling, stop and tool_choice](chat-request-params.md) | Anthropic -> Chat request | new | `repro/internal/protocol/request/` |
| 12 | medium | [recordings keep client credentials](recording-client-credentials.md) | recording | new, minor | code reading |
| 15 | medium | [Responses developer messages become user messages](responses-developer-message-as-user.md) | Responses -> Chat request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| 16 | medium | [`model_context_window_exceeded` and `refusal` reported as normal completions](anthropic-abnormal-stop-reasons.md) | Anthropic -> Chat non-stream, Anthropic -> Responses stream and non-stream | new | `repro/internal/protocol/{stream,nonstream}/` |
| 17 | medium | [Chat tool calls without an id stay without one](chat-tool-call-empty-id.md) | Chat -> Anthropic stream and non-stream, Chat -> Responses non-stream | new | `repro/internal/protocol/{stream,nonstream}/`, `repro/internal/protocolserver/` |
| 18 | medium | [Chat -> Anthropic requests drop developer messages, parameterless tools, tool_choice modes](chat-anthropic-request-coverage.md) | Chat -> Anthropic request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| 19 | medium | [Chat -> Responses requests drop max_completion_tokens, zero sampling values, parallel_tool_calls](chat-responses-request-params.md) | Chat -> Responses request | new | `repro/internal/protocol/request/`, `repro/internal/protocolserver/` |
| 20 | medium | [upstream errors not translated to the client's protocol](upstream-errors-not-translated.md) | error responses, all converted paths | new (related: #2052) | `repro/internal/protocolserver/` |
| 24 | medium | [parameterless tool calls stream arguments `""`](parameterless-tool-stream-arguments.md) | Anthropic -> Chat and Anthropic -> Responses streams | new (related: #1840) | `repro/internal/protocol/stream/` |
| 26 | medium | [a refusal becomes `content: null`](responses-anthropic-nonstream-refusal.md) | Responses -> Anthropic non-stream | new (related: #912) | `repro/internal/protocol/nonstream/` |
| 33 | medium | [tool_choice "none" becomes auto](responses-anthropic-tool-choice-none.md) | Responses -> Anthropic request | new (found by the go2lean M2 agent) | `repro/internal/protocol/request/` |
| 38 | medium | [count_tokens estimate ignores tool_use, tool_result, document and image blocks](count-tokens-ignores-tool-blocks.md) | Anthropic count_tokens, providers without a native endpoint | new | `repro/internal/protocol/token/` |
| 39 | medium | [Gemini tool schemas: exclusive bounds overwrite minimum/maximum at random, nested schemas not filtered](gemini-schema-exclusive-bounds.md) | Chat -> Gemini (Google, Poe) request | new | `repro/internal/protocol/ops/` |
| 11 | low | [Chat -> Anthropic stream opens a block while another is open](chat-anthropic-interleaved-blocks.md) | Chat -> Anthropic stream | found by go2lean, minor | `benchmark.md` |
| 27 | low | [one assistant turn becomes two assistant messages](responses-chat-split-assistant-turn.md) | Responses -> Chat request | new | `repro/internal/protocol/request/` |
| 28 | low | [reasoning summaries dropped without streaming](responses-anthropic-nonstream-reasoning-summary.md) | Responses -> Anthropic non-stream | new (related: #1863) | `repro/internal/protocol/nonstream/` |
| 29 | low | [images in tool output dropped](responses-anthropic-tool-output-image.md) | Responses -> Anthropic request | new | `repro/internal/protocol/request/` |
| 30 | low | [tool `strict` flag lost](chat-responses-tool-strict.md) | Chat -> Responses, Responses -> Chat requests | new | `repro/internal/protocol/request/` |
| 31 | low | [stream input usage depends on which fields message_delta repeats](usage-anthropic-delta-cache-creation.md) | usage accumulation, Anthropic provider streams | new, minor | `repro/internal/protocol/usage/` |
| 32 | low | [`ParseAnthropicResult` keeps only the last text block](sse-parse-anthropic-last-text.md) | test infrastructure (`sse` parsers) | new, minor | `repro/internal/protocol/sse/` |
| 34 | low | [Gemini function calling mode NONE becomes auto](gemini-tool-choice-none.md) | Gemini -> Chat, Gemini -> Anthropic request (no gateway caller) | new, latent (found by the go2lean M2 agent) | `repro/internal/protocol/request/` |
| 40 | low | [Gemini thinking config replaces `extra_body`](gemini-thinking-extra-body-overwritten.md) | Chat -> Gemini request (client `extra_body` already dropped at parsing) | new, latent | `repro/internal/protocol/ops/` |
| 41 | low | [recordings of Anthropic streams lose input and cache usage](recording-anthropic-stream-usage.md) | recording, Anthropic streams | new, minor | `repro/internal/protocol/assembler/` |
| 42 | low | [`ResponsesAssembler.ToolCalls()` arguments always empty](responses-assembler-tool-call-arguments.md) | Responses stream assembler (no gateway caller) | new, latent | `repro/internal/protocol/assembler/` |
| 43 | low | [`StreamTokenCounter.Reset` keeps the upstream usage](stream-counter-reset-upstream.md) | streaming token counter (no gateway caller) | new, latent | `repro/internal/protocol/token/` |

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
(MiniMax-M3); their repro tests were written afterwards. Findings 35-43
(from `anthropic-metadata-empty-account-panic` to
`stream-counter-reset-upstream`) were found by hand again, in a fourth round
of code reading over the vendor request transforms (Anthropic model and
metadata, Gemini thinking and tool schemas), the stream assemblers (the
Codex non-stream path, recordings) and the token counters, and confirmed
with `go test`; their pages follow the same form.

Not bugs (the converters' intended repairs, recorded so they are not
reported): orphan tool calls get an "aborted" result, an unmatched
function_call_output becomes a user message (both in the Responses and
Anthropic round trips); a refusal is carried as text to Anthropic.
