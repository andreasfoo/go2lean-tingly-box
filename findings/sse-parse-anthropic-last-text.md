# `sse.ParseAnthropicResult` keeps only the last text block (new, not reported; test infrastructure)

Path: tingly-box's result parsers, used by its own test harnesses
(`internal/protocoltest/testenv.go:677`, `internal/protocoltest/duo_routing.go:442`,
`vmodel/vmodeltest/client.go:153`), not by the gateway.

Where (tingly-box `b788a3b`): `internal/protocol/sse/parser.go:145`
(`ParseAnthropicResult`) assigns `r.Content` for each text block, so the
last one wins; `:103` (`ParseOpenAIResponsesResult`) does the same for the
`output_text` parts of a message. `AssembleAnthropicStream` (`:318`)
concatenates the text of all blocks.

Trigger: an Anthropic message with more than one text block (text around a
tool_use, citations, server tools), or a Responses message with several
`output_text` parts.

Observed: for content `[text "Hello, ", text "world."]`
`ParseAnthropicResult` gives `"world."`, `AssembleAnthropicStream` on the
same message streamed gives `"Hello, world."`.

Expected: `"Hello, world."` from both: the same message parses to the same
text, streamed or not.

Realism: low, test infrastructure only. A protocol test that compares the
non-streamed with the streamed result, or checks the text of a multi-block
reply, sees a difference that is not in the gateway, or misses one that is.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ParseAnthropicKeepsAllTextBlocks`
(`repro/internal/protocol/sse/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Anthropic message, `ParseAnthropicResult` of the message and
`AssembleAnthropicStream` of its event stream give the same content: the
concatenation of the text of all text blocks in order.

Target function: `sse.ParseAnthropicResult` (and
`sse.ParseOpenAIResponsesResult` for Responses).
