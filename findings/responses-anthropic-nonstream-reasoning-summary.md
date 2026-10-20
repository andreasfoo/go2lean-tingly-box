# Responses -> Anthropic without streaming: reasoning summaries dropped (new, not reported)

Path: Anthropic client -> Responses provider, not streaming.

Where (tingly-box `b788a3b`):
`internal/protocol/nonstream/openai_to_anthropic.go:201-207`
(`HandleResponsesToAnthropicBeta`). Thinking blocks are made only from
`content[]` parts of type `reasoning_text`; a reasoning item's `summary[]`
is never read. The thinking blocks it does make are appended after the text
and tool_use blocks (`:184-199` run first), while Anthropic puts thinking
first. The streaming converter makes a thinking block from the summary
(`stream/openai_responses_to_anthropic_converter.go:280`).

Trigger: a Responses reasoning item with summary text, which OpenAI's
reasoning models return when `reasoning.summary` is requested (the Codex
provider requests `"auto"`, `internal/protocol/ops/request_openai_codex.go:84`).

Observed: for output `[reasoning {summary "Compared both options."},
message "Option A."]` the content is `[{"text":"Option A.","type":"text"}]`.

Expected: `[{"type":"thinking","thinking":"Compared both options.",…},
{"type":"text","text":"Option A."}]`, as the streaming path gives.

Realism: an Anthropic client that does not stream (scripts, SDK `create`
calls, side requests) on a gpt-5 / o-series / Codex provider loses all of the
model's reasoning; the same request streamed gets it.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20;
related, not the same: #1863, reasoning field spellings on the Chat ->
Anthropic path).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_NonStreamReasoningSummaryIsThinking`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Responses response, the converted Anthropic message has one
thinking block per reasoning item that has summary or reasoning text, with
that text, and the thinking blocks come before the text and tool_use blocks
of the same turn.

Target function: `nonstream.HandleResponsesToAnthropicBeta`.
