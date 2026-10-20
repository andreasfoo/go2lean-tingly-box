# Responses -> Anthropic stream: an empty thinking block stays open across the answer (new, not reported)

Path: Anthropic client -> Responses provider, streaming.

Where (tingly-box `b788a3b`),
`internal/protocol/stream/openai_responses_to_anthropic_converter.go`:

- `response.output_item.added` with a `reasoning` item (`:316-325`) opens a
  thinking block and sends a thinking delta read from the event as if it
  were a `reasoning_text.delta` (always `""`).
- `response.output_item.done` (`:397`) does nothing ("handled by respective
  .done events above").
- The block is closed only by `response.reasoning_text.done` (`:272-278`),
  which OpenAI's hosted reasoning models do not stream (they stream
  summaries), or by `finalize` at the end of the stream.
- Summary text opens a second thinking block of its own
  (`response.reasoning_summary_text.delta`, `:280`).

Trigger: a Responses stream with a reasoning output item, which every
reasoning model (gpt-5, o-series, the Codex models) sends before its answer,
with or without `reasoning.summary`.

Observed (block starts and stops the client receives):

- no summary: `start 0 thinking, start 1 text, stop 1, stop 0`; block 0
  has no thinking text and ends with a made-up signature after the answer;
- with a summary: `start 0 thinking, start 1 thinking, stop 1, start 2
  text, stop 2, stop 0`.

Expected: one block open at a time (each `content_block_start` after the
previous block's `content_block_stop`), and no empty thinking block: the
reasoning item is its summary's thinking block, or nothing.

Realism: high for Claude Code on a gpt-5 or Codex provider: every turn opens
with this sequence. The Anthropic SDKs assemble by index and cope with the
interleaving, but the message they build starts with an empty thinking block
(two thinking blocks with a summary), which Claude Code then sends back in
its history; a client that follows the API's one-block-at-a-time order (or
renders thinking as it streams) sees thinking that never ends until the
message does. Same class as `chat-anthropic-interleaved-blocks.md`, in
another converter.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ReasoningItemNoSummaryOneOpenBlock` and
`TestGo2LeanRepro_ReasoningItemWithSummaryOneOpenBlock`
(`repro/internal/protocol/stream/zz_go2lean_repro_hunt3_test.go`).

Fix direction: open the thinking block lazily on the first reasoning text or
summary delta, and close it on `response.output_item.done` of the reasoning
item.

## General property

For every Responses event stream, the Anthropic events the converter emits
have at most one content block open at a time (every `content_block_start`
comes after the `content_block_stop` of the block before it), and every
thinking block it starts carries non-empty thinking text before its stop.

Target function: `(*responsesToAnthropicConverter).Next` (via
`NewOpenAIResponsesToAnthropicConverter`).
