package token

// Reproductions on the local token counters, found by hand (code reading,
// round 4). Each test asserts the behaviour a caller expects: it fails while
// the bug is there.

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
)

// findings/count-tokens-ignores-tool-blocks.md: count_tokens for a
// provider without a native endpoint counts only text and thinking blocks,
// so a 20 KB tool_result adds nothing.
func TestGo2LeanRepro_CountTokensIncludesToolResult(t *testing.T) {
	big := strings.Repeat("package main\nfunc main() { println(\"hello world\") }\n", 400)
	base := []anthropic.BetaMessageParam{
		anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("read main.go")),
		{Role: anthropic.BetaMessageParamRoleAssistant, Content: []anthropic.BetaContentBlockParamUnion{
			anthropic.NewBetaToolUseBlock("toolu_1", map[string]any{"file_path": "/src/main.go"}, "Read"),
		}},
	}
	with := append(append([]anthropic.BetaMessageParam{}, base...),
		anthropic.NewBetaUserMessage(anthropic.NewBetaToolResultBlock("toolu_1", big, false)))
	n0, _ := CountBetaTokensViaTiktoken(&anthropic.BetaMessageCountTokensParams{Model: "gpt-5", Messages: base})
	n1, _ := CountBetaTokensViaTiktoken(&anthropic.BetaMessageCountTokensParams{Model: "gpt-5", Messages: with})
	if n1-n0 < 1000 {
		t.Errorf("adding a %d-byte tool_result changed the count from %d to %d; want an increase of thousands of tokens", len(big), n0, n1)
	}
}

// findings/stream-counter-reset-upstream.md: Reset clears the local counts
// but not the upstream usage, so GetCounts still reports the previous
// stream's usage.
func TestGo2LeanRepro_StreamCounterResetClearsUpstream(t *testing.T) {
	c, err := NewStreamTokenCounter()
	if err != nil {
		t.Fatal(err)
	}
	_ = c.ConsumeOpenAIChunk(&openai.ChatCompletionChunk{Usage: openai.CompletionUsage{PromptTokens: 100, CompletionTokens: 50}})
	c.Reset()
	if i, o := c.GetCounts(); i != 0 || o != 0 {
		t.Errorf("GetCounts after Reset = (%d, %d), the usage consumed before the reset; want (0, 0)", i, o)
	}
}
