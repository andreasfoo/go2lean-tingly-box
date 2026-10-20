# Finding 4: reasoning_content empty without streaming (new, not reported)

Anthropic -> Chat, non-streaming: for a `thinking` block the converter
fills `reasoning_content` from `block.Text` (empty for thinking blocks)
instead of `block.Thinking` (`internal/protocol/nonstream/anthropic.go:45-46`).
The streaming converter returns the thinking text, so the same upstream
answer reads differently with and without `stream: true`.

Spec: `SpecAnthropicToOpenAIConverterProcessEventStreamReasoningMatches`
(`specs/internal/protocol/stream/`). Found at trial 1335, confirmed by go
test; a second check on a decoded API response agrees. PoC in
`poc/round3/`. Tracker search for `reasoning_content` (2026-10-19): #1863
covers the Chat -> Anthropic direction only.
