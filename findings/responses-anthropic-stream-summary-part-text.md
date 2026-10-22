# Finding 85: Responses -> Anthropic streams drop reasoning summary text carried by `reasoning_summary_part.added` (new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

The `response.reasoning_summary_part.added` branch of
`responsesToAnthropicConverter.processEvent`
(`internal/protocol/stream/openai_responses_to_anthropic_converter.go:423-436` at `bc460f6`) opens a thinking block and emits the part's
text only when `Part.Type == "text"`. The Responses API (and openai-go's
`ResponseReasoningSummaryPartAddedEventPart`) types a summary part
`"summary_text"`, so the branch never runs, and text carried by the part
itself is dropped.

Path: an Anthropic client with a Responses provider, streaming, with
reasoning summaries. Latent: OpenAI sends the part empty and streams the
text in `response.reasoning_summary_text.delta` events, which the
converter handles; only a provider or proxy that puts the text in the part
(as `reasoning_summary_part.done` does) loses it.

Trigger: `output_item.added` (reasoning), `reasoning_summary_part.added`
with `part: {"type":"summary_text","text":"Plan the call."}`,
`reasoning_summary_part.done`, `output_item.done`, `response.completed`.

Observed: no thinking block; the summary is lost.

Expected: a thinking block with `"Plan the call."`.

How it was found: as [finding 83](responses-anthropic-stream-tool-then-text-end-turn.md),
by the agent's own Go runs while triaging go2lean's sequence specs.

Repro: `TestGo2LeanRepro_ResponsesToAnthropicSummaryPartText`
(`repro/internal/protocol/stream/zz_go2lean_repro_seq_test.go`).
