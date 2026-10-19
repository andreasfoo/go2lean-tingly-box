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

Not bugs (the converters' intended repairs, recorded so they are not
reported): orphan tool calls get an "aborted" result, an unmatched
function_call_output becomes a user message (both in the Responses and
Anthropic round trips); a refusal is carried as text to Anthropic.
