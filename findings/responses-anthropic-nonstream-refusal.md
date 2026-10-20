# Finding 26: Responses -> Anthropic without streaming: a refusal becomes `content: null` (new, not reported)

Path: Anthropic client -> Responses provider, not streaming.

Where (tingly-box `b788a3b`), `internal/protocol/nonstream/openai_to_anthropic.go`
(`HandleResponsesToAnthropicBeta`, `:171`):

- `:216-223`: for a content part of type `refusal` the converter reads
  `content.Text`, which is always empty; the refusal text is in
  `content.Refusal`. So no block is added and the stop reason is not set to
  `refusal`.
- `:225`: `wire.Content = contentBlocks` with a nil slice, which marshals as
  `"content": null`.

The streaming converter carries the refusal as text
(`stream/openai_responses_to_anthropic_converter.go:298-311`).

Trigger: a Responses output message whose content is a `refusal` part (and,
for the `null`, any response whose output yields no block).

Observed: `{"id":"resp_1","type":"message","role":"assistant","content":null,"model":"m","stop_reason":"end_turn",…}`.

Expected: `content` `[{"type":"text","text":"I can't help with that."}]`
and `stop_reason` `"refusal"` (a refusal is carried to Anthropic as text,
as the streaming path does); `content` always an array.

Realism: OpenAI models refuse with a `refusal` part, more often with
structured outputs. The client loses the refusal text and sees a normal
`end_turn`; a client that iterates `content` (the Anthropic SDKs type it as
a list) fails on `null`.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20;
related, not the same: #912, where content_filter -> end_turn is
intentional).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ResponsesRefusalReachesAnthropicClient`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Responses response, the converted Anthropic message has an array
`content`, and every non-empty text of the response's message parts
(`output_text` text and `refusal` refusal) is the text of one of its text
blocks; when a refusal part is present the stop reason is `refusal`.

Target function: `nonstream.HandleResponsesToAnthropicBeta`.
