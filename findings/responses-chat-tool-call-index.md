# Finding 14: Chat tool call index taken from the Responses output index (new, not reported)

Path: Chat client -> Responses provider, streaming.

Where (tingly-box `b788a3b`):
`internal/protocol/stream/openai_responses_to_chat_converter.go:130`
(`response.output_item.added`: `index := int(evt.OutputIndex)`), and the
same fallback at `:147`, `:158`, `:168` and in `emitCompletedOutput` at
`:250` (`index := outputIndex`). The Chat `tool_calls[].index` is set to the
Responses `output_index`, which counts every output item (reasoning,
messages, calls), not only function calls.

Trigger: a Responses stream in which a function call is not the first
output item; a reasoning model emits its `reasoning` item at
`output_index` 0 and its first `function_call` at 1.

Observed: the only tool call streams with `tool_calls[].index` 1.

Expected: index 0. Chat tool call indexes number the tool calls of the
choice, from 0 without gaps.

Realism: every reasoning model on the Responses API (gpt-5, o-series, the
Codex models) emits a reasoning item before its tool calls, so a Chat
client of tingly-box gets index >= 1 on nearly every tool call from them.
openai-python's `accumulate_delta` (used by `client.chat.completions.stream`)
does `acc[index]` and, on `IndexError`, `insert(index, entry)`: with index
1 on an empty list each delta is inserted as a new entry, so one call
becomes two (the first with the name and no arguments, the second nameless
with the arguments). Clients that index an array by it (as JS clients
often do) get a hole at 0.

The package's golden test pins the bug:
`TestResponsesToChatConverter_GoldenSequence`
(`openai_responses_to_chat_golden_test.go:141`) asserts
`assert.Equal(t, 1, tc.Index)` for a call after a reasoning item.

Status: **new** (tracker search 2026-10-20 for `tool_calls index`,
`output_index`: nothing; #1923 is the Anthropic -> Chat stream putting all
calls at index 0, a different converter).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ToolCallIndexAfterReasoningItem`
(`repro/internal/protocol/stream/zz_go2lean_repro_hunt2_test.go`).

Fix direction: keep a counter of function calls seen and map each item id to
the next counter value; fix the golden test's expectation to 0.

## General property

For every Responses event stream, the Chat `tool_calls[].index` values the
converter emits for its function calls are 0, 1, …, n-1 in the order the
calls start, whatever other output items come before or between them; all
chunks of one call carry the same index.

Target function: `(*responsesToChatConverter).Next` (via
`NewOpenAIResponsesToChatConverter`).
