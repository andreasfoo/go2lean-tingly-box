package nonstream

// Reproduction of a go2lean finding on the non-streaming converters. The
// test asserts the behaviour a client expects: it fails while the bug is
// there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// findings/reasoning-content-nonstream.md
func TestGo2LeanRepro_ThinkingReachesReasoningContent(t *testing.T) {
	var bm anthropic.BetaMessage
	raw := `{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5",
		"content":[{"type":"thinking","thinking":"Let me add the numbers.","signature":"sig"},
		           {"type":"text","text":"The answer is 4."}],
		"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":20}}`
	if err := json.Unmarshal([]byte(raw), &bm); err != nil {
		t.Fatal(err)
	}
	out := ConvertAnthropicBetaToOpenAIChat(&bm, "m")
	if got := out.Choices[0].Message.ReasoningContent; got != "Let me add the numbers." {
		t.Errorf("reasoning_content %q, want the thinking text (the streaming converter returns it)", got)
	}
}
