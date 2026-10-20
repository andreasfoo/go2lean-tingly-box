package nonstream

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3"
)

// findings/chat-anthropic-reasoning-null.md (non-streaming): the Chat ->
// Anthropic message conversion builds the thinking block from the raw JSON
// of reasoning_content, so the client reads the JSON string literal (quotes
// and escapes) and a null reasoning_content becomes the thinking "null".
func TestGo2LeanRepro_NonStreamThinkingIsPlainText(t *testing.T) {
	for _, tc := range []struct{ reasoning, want string }{
		{`"Step 1.\nStep 2."`, "Step 1.\nStep 2."},
		{`null`, ""},
	} {
		var chat openai.ChatCompletion
		raw := `{"id":"c","object":"chat.completion","created":1,"model":"deepseek-reasoner","choices":[{"index":0,
			"message":{"role":"assistant","content":"Done.","reasoning_content":` + tc.reasoning + `},"finish_reason":"stop"}]}`
		if err := json.Unmarshal([]byte(raw), &chat); err != nil {
			t.Fatal(err)
		}
		msg, err := ConvertOpenAIChatToAnthropicBeta(&chat, "m")
		if err != nil {
			t.Fatal(err)
		}
		thinking, found := "", false
		for _, b := range msg.Content {
			if b.Type == "thinking" {
				thinking, found = b.Thinking, true
			}
		}
		if tc.want == "" && found {
			t.Errorf("reasoning_content %s: a thinking block %q, want none", tc.reasoning, thinking)
		}
		if tc.want != "" && thinking != tc.want {
			t.Errorf("reasoning_content %s: thinking %q, want %q", tc.reasoning, thinking, tc.want)
		}
	}
}
