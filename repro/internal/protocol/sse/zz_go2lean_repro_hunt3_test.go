package sse

// Reproduction on the SSE/result parsers, found by hand (code reading,
// round 3). The test asserts the behaviour a caller expects: it fails while
// the bug is there.

import (
	"encoding/json"
	"testing"
)

// findings/sse-parse-anthropic-last-text.md: the same two-text-block
// Anthropic message, non-streamed and streamed, must parse to the same text.
func TestGo2LeanRepro_ParseAnthropicKeepsAllTextBlocks(t *testing.T) {
	nonstream := `{"type":"message","role":"assistant","model":"m","stop_reason":"end_turn",
	 "content":[{"type":"text","text":"Hello, "},{"type":"text","text":"world."}],
	 "usage":{"input_tokens":3,"output_tokens":4}}`
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(nonstream), &m); err != nil {
		t.Fatal(err)
	}
	parsed := ParseAnthropicResult(m)

	assembled := AssembleAnthropicStream([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"model":"m","usage":{"input_tokens":3,"output_tokens":0}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello, "}}`,
		`data: {"type":"content_block_stop","index":0}`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"world."}}`,
		`data: {"type":"content_block_stop","index":1}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":4}}`,
		`data: {"type":"message_stop"}`,
	})
	if parsed.Content != "Hello, world." || assembled.Content != "Hello, world." {
		t.Errorf("ParseAnthropicResult content %q, AssembleAnthropicStream content %q, want both \"Hello, world.\"", parsed.Content, assembled.Content)
	}
}
