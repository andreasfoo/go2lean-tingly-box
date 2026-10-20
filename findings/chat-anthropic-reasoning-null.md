# DeepSeek-style reasoning_content breaks thinking blocks for Anthropic clients (new, not reported)

DeepSeek's reasoning models (and servers that copy their format) stream
the reasoning with `"content":null` and then the answer with
`"reasoning_content":null` in every chunk:

```
{"delta":{"content":null,"reasoning_content":"Think."}}
{"delta":{"content":"It","reasoning_content":null}}
{"delta":{"content":" works.","reasoning_content":null}}
```

An Anthropic client (Claude Code) routed to such a provider through an
OpenAI Chat provider goes through the Chat -> Anthropic converters, which
treat the *presence* of the key as reasoning:

- **Streaming** (`internal/protocol/stream/openai_to_anthropic_converter.go`,
  delta-extras loop in `processChunk`; used by
  `stage/anthropicbridge/stream.go`): the first answer chunk closes the
  thinking block and opens the text block (thinking index reset to -1). The
  next chunk's `reasoning_content: null` opens a new thinking block while
  the text block is still open, and the text deltas keep going to the
  earlier block. Through the real gateway (`/tingly/claude_code/v1/messages`
  with a Chat provider serving the chunks above) the client receives:

  ```
  content_block_start 0 thinking … content_block_stop 0
  content_block_start 1 text   content_block_delta 1
  content_block_start 2 thinking            <- block 1 still open
  … content_block_delta 1 … content_block_stop
  ```

  The assembled message is `[thinking, text, thinking("")]`: an empty
  thinking block with a made-up signature after the answer, and an event
  order the Anthropic API never produces (one open block at a time).
- **Non-streaming** (`internal/protocol/nonstream/openai_to_anthropic.go`,
  both message conversions): the thinking block is built from
  `fmt.Sprintf("%s", thinking.Raw())`, the raw JSON, so the client reads
  the JSON string literal — `"Step 1.\nStep 2."` with the quotes and a
  literal backslash-n — and `reasoning_content: null` (or `""`) becomes a
  thinking block reading `null` (or `""`). The block is also placed after
  the text.

The streaming fix is to open a thinking block only for non-empty text (as
the delta itself already does) and to never reopen one after the answer
started; the non-streaming fix is to decode the string (the open #1863
suggests reusing `stream.ExtractReasoningText`, which would do that).

How it was found: bug class 4/9 of `hunting-similar-tools.md` (empty or
mismatched blocks in Chat -> Anthropic streams, seen in litellm and
new-api), checked with an event-order checker over Chat chunk shapes. The
DeepSeek shape is from a raw stream quoted in karthink/gptel#669.

Repro: `TestGo2LeanRepro_NullReasoningOpensNoThinkingBlock`
(`repro/internal/protocol/stream/zz_go2lean_repro_c2a_test.go`) and
`TestGo2LeanRepro_NonStreamThinkingIsPlainText`
(`repro/internal/protocol/nonstream/zz_go2lean_repro_c2a_test.go`).

Tracker: #1863 (open) is about the other spellings (`reasoning`,
`reasoning_details`) not being read at all; it does not mention the null
handling, the raw-JSON text, or the block order.
