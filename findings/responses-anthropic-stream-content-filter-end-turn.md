# Finding 81: Responses -> Anthropic streams report a content-filtered answer as `end_turn` (new, not reported)

Severity: **low** (see [the findings table](README.md)).

`responsesToAnthropicConverter.processEvent`
(`internal/protocol/stream/openai_responses_to_anthropic_converter.go:405-415`)
maps `response.incomplete` with `incomplete_details.reason:
"content_filter"` to `stop_reason: "end_turn"`, a normal completion. The
non-stream converter `HandleResponsesToAnthropicBeta`
(`nonstream/openai_to_anthropic.go:208-213`) gives `refusal` for the same
response. The stream mapping dates from #1132; the non-stream one moved to
`refusal` later, in `4721cf6c4` (#1504).

Path: an Anthropic client with a Responses provider (OpenAI, Azure OpenAI
with its content filter), streaming: `stage/anthropicbridge/responses_stream.go:23`.
The cut-short, filtered answer looks complete to the client.

Trigger: the terminal event
`{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"content_filter"},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Here is how"}]}],…}}`.

Observed: `message_delta.stop_reason` `"end_turn"`; without streaming,
`"refusal"`.

Expected: `"refusal"` on both paths (Anthropic's stop reason for a policy
stop).

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecResponsesToAnthropicConverterProcessEventIssue1132IncompleteAgreesNonStream`
was refuted, `go test` confirmed the counterexample, and the repro test was
written afterwards.

Repro: `TestGo2LeanRepro_ResponsesToAnthropicStreamContentFilterAgrees`
(`repro/internal/protocol/stream/zz_go2lean_repro_issues1_test.go`).
