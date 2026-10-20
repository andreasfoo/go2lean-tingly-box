# Finding 11: Chat -> Anthropic stream opens a block while another is open (found by go2lean; minor)

Severity: **low** (see [the findings table](README.md)).

The Anthropic streaming API sends content blocks one after another: a
`content_block_start`, its deltas, its `content_block_stop`, then the next
block. The Chat -> Anthropic stream converter
(`internal/protocol/stream/openai_to_anthropic_converter.go`, `processChunk`)
closes only an open *text, thinking or refusal* block before starting a new
one (`emitCloseOpenBlock`); a tool_use block stays open until the end of
the stream. So:

- parallel tool calls (`tool_calls[].index` 0 and 1) start block 1 while
  block 0 is open, and their argument deltas interleave;
- text (or a refusal) after a tool call starts a text block while the tool
  block is open.

Clients that assemble by index (the Anthropic SDKs) cope; a client that
expects the API's order (one open block) does not. Minor, but it is a
departure from the protocol the converter emulates.

How it was found: by go2lean, blind, on the general property
`SpecOpenAIToAnthropicConverterProcessChunkOneOpenBlock` ("whatever deltas a
Chat stream sends, one content block is open at a time and every delta goes
to it"): found at trial 683 (a refusal, then tool calls at indexes 1 and
-5), confirmed by `go test`. The first M3 run was inconclusive because the
harness dropped an empty map literal (an empty map is not the zero value);
fixed in go2lean, then confirmed.

Where (tingly-box `b788a3b`, unchanged on main `bc460f6`):
`internal/protocol/stream/openai_to_anthropic_converter.go:400-423`
(`emitCloseOpenBlock`) closes a thinking, reasoning-summary, refusal or text
block, never a tool_use block; tool_use blocks are stopped only at the end
(`emitStopEvents`, `:425`).

Status: found by go2lean, minor (tracker search 2026-10-20 over all issues
and PRs for `content_block_start`, `interleaved`, `block still open`:
nothing; #1502 and #1505, both merged, fixed interleaved blocks in the
Anthropic -> Responses direction, not this one). Reproduces on main
`bc460f6`.

Evidence: `TestGo2LeanRepro_ParallelToolCallsOneOpenBlock` (two parallel
tool calls in the order OpenAI streams them) and
`TestGo2LeanRepro_TextAfterToolCallOneOpenBlock`
(`repro/internal/protocol/stream/zz_go2lean_repro_blocks_test.go`); both
fail on `b788a3b` and `bc460f6` with "block 1 (tool_use|text) starts while
block 0 is open".
