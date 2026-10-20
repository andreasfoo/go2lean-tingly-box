# Finding 27: Responses -> Chat: one assistant turn becomes two assistant messages (new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Responses client -> Chat provider (request conversion; streaming or
not).

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_responses_to_chat.go:110-184`
(`ConvertResponsesInputToMessages`). An assistant `message` item (or output
message) becomes an assistant message of its own (`:159-175`); the
`function_call` items after it are collected and flushed as a second
assistant message with `content: ""` and the `tool_calls` (`:117-142`). The
comment above the function (`:101-109`) says the conversion "only has to
preserve order".

Trigger: a Responses history in which an assistant message is followed by
function calls of the same turn: the model's preamble ("I'll list the
directory.") and then its tool call.

Observed:
`[{"role":"user",…},{"content":"I'll list the directory.","role":"assistant"},{"content":"","tool_calls":[{"id":"call_1",…}],"role":"assistant"},{"content":"a.txt","tool_call_id":"call_1","role":"tool"}]`.

Expected: one assistant message with the text as `content` and the calls as
`tool_calls`, as the Chat API itself returns such a turn.

Realism: high for Codex on a Chat provider: gpt-5 and the Codex models send
a preamble before most tool calls, and Codex replays its history with each
request, so nearly every request carries such pairs (the gateway's
`PreprocessInputData` flattens the `output_text` parts into the string form
used in the test; the output-message form splits the same way). Providers
that require alternating roles reject the request; others see two turns
where the model made one, with an empty-content assistant message between.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ResponsesAssistantTurnIsOneChatMessage`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Responses input, the converted Chat messages never contain two
consecutive assistant messages: the text of an assistant message and the
function calls that directly follow it are one assistant message, with all
of the text and all of the calls in order.

Target function: `request.ConvertOpenAIResponsesToChat` (via
`ConvertResponsesInputToMessages`).
