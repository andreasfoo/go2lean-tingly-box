# Finding 54: Anthropic -> Responses: Claude's thinking dropped (found by go2lean, blind specs; new, not reported)

Severity: **low** (see [the findings table](README.md)).

Path: Responses client (Codex CLI and other Responses clients) -> Anthropic
provider with thinking on, response conversion, non-stream; the stream
converter, by reading, too.

Where (tingly-box `b788a3b`):

- non-stream: `ConvertAnthropicBetaToResponsesWire`
  (`internal/protocol/nonstream/openai_to_responses.go:125-201`) emits text
  blocks (`:137`) and tool_use blocks (`:161`) and ignores `thinking`
  blocks: no reasoning item, no summary. Live via
  `internal/protocol/stage/responsesbridge/bridge.go:102`.
- stream, by reading: the converter
  (`internal/protocol/stream/anthropic_beta_to_openai_responses_converter.go`)
  handles `text` and `tool_use` block starts (`:176`, `:203`) and
  `text_delta` / `input_json_delta` (`:230`, `:245`) only; thinking blocks
  and `thinking_delta` are skipped.

Trigger: a Claude answer `[thinking "The user wants the capital; it is
Paris.", text "Paris."]`.

Observed: the Responses output holds only the message item.

Expected: a `reasoning` output item before the message, carrying the
thinking as a summary (and the signature as `encrypted_content`, so a
later turn can send it back).

Realism: low. Codex CLI or another Responses client on a Claude provider
with a reasoning effort set loses the reasoning display; the answer itself
is intact. `responses-anthropic-nonstream-reasoning-summary.md` (#28) is
the reverse direction; `anthropic-abnormal-stop-reasons.md` (#16) covers
this function's stop reasons.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `reasoning item`, `thinking` with `responses`: nothing; #1897,
closed, is thinking effort on the request side). Reproduces on main
`bc460f6`.

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)):
`SpecConvertAnthropicBetaToResponsesWireThinking` was refuted, `go test`
confirmed the counterexample, and triage found no earlier finding for it.
The stream half was found by reading; the repro test was written
afterwards with a JSON-parsed message.

Evidence: `TestGo2LeanRepro_AnthropicThinkingReachesResponsesClient`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_blind_test.go`).

## General property

For every Anthropic message, the text of each thinking block appears in a
`reasoning` item of the Responses output, placed before the message and
function-call items that follow it.

Target function: `nonstream.ConvertAnthropicBetaToResponsesWire`.
