package nonstream

// Reproductions on the non-streaming converters for findings 44-56, found by
// go2lean from specs written blind (a Claude agent that had never seen the
// findings wrote them from the package source and go2lean's agent prompt;
// go2lean searched them and `go test` confirmed the counterexamples). The
// tests were written afterwards, with realistic JSON-parsed responses. Each
// test asserts the behaviour a client expects: it fails while the bug is
// there.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

func reproBlJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func reproBlChatCompletion(t *testing.T, body string) *openai.ChatCompletion {
	t.Helper()
	var c openai.ChatCompletion
	if err := json.Unmarshal([]byte(body), &c); err != nil {
		t.Fatal(err)
	}
	return &c
}

// findings/chat-stop-with-tool-calls-end-turn.md: an OpenAI-compatible
// server that reports finish_reason "stop" for a tool call.
func TestGo2LeanRepro_ChatStopWithToolCallsIsToolUse(t *testing.T) {
	chat := reproBlChatCompletion(t, `{"id":"c","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,
	 "message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]},
	 "finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	msg, err := ConvertOpenAIChatToAnthropicBeta(chat, "m")
	if err != nil {
		t.Fatal(err)
	}
	hasToolUse := false
	for _, b := range msg.Content {
		if b.Type == "tool_use" {
			hasToolUse = true
		}
	}
	if !hasToolUse {
		t.Fatalf("tool call not carried: %s", msg.RawJSON())
	}
	if msg.StopReason != anthropic.BetaStopReasonToolUse {
		t.Errorf("a message with a tool_use block ends with stop_reason %q; clients that run tools on \"tool_use\" end the turn instead", msg.StopReason)
	}
}

// findings/gemini-thoughts-tokens-dropped.md: a Gemini 2.5 thinking model
// answer, with the prompt partly cached.
const reproBlGeminiThinkingResponse = `{"candidates":[{"content":{"role":"model","parts":[{"text":"Paris."}]},"finishReason":"STOP","index":0}],
 "usageMetadata":{"promptTokenCount":1000,"cachedContentTokenCount":900,"candidatesTokenCount":50,"thoughtsTokenCount":400,"totalTokenCount":1450},
 "modelVersion":"gemini-2.5-pro"}`

func TestGo2LeanRepro_GeminiThoughtsTokensCountedAsOutput(t *testing.T) {
	var g genai.GenerateContentResponse
	if err := json.Unmarshal([]byte(reproBlGeminiThinkingResponse), &g); err != nil {
		t.Fatal(err)
	}
	if g.UsageMetadata == nil || g.UsageMetadata.ThoughtsTokenCount != 400 {
		t.Fatalf("fixture not parsed: %s", reproBlJSON(t, g.UsageMetadata))
	}
	beta := HandleGoogleToAnthropicBeta(&g, "m")
	if out := beta.Usage.OutputTokens; out != 450 {
		t.Errorf("Anthropic client: output_tokens %d for 50 candidate + 400 thought tokens, want 450", out)
	}
	chat := reproBlJSON(t, HandleGoogleToOpenAI(&g, "m"))
	if c := gjson.Get(chat, "usage.completion_tokens").Int(); c != 450 {
		t.Errorf("Chat client: completion_tokens %d for 50 candidate + 400 thought tokens, want 450 (usage %s)", c, gjson.Get(chat, "usage").Raw)
	}
}

// findings/anthropic-responses-thinking-dropped.md: Claude with extended
// thinking answering a Responses client.
func TestGo2LeanRepro_AnthropicThinkingReachesResponsesClient(t *testing.T) {
	var m anthropic.BetaMessage
	if err := json.Unmarshal([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5",
	 "content":[{"type":"thinking","thinking":"The user wants the capital; it is Paris.","signature":"sig"},{"type":"text","text":"Paris."}],
	 "stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":30}}`), &m); err != nil {
		t.Fatal(err)
	}
	w := reproBlJSON(t, ConvertAnthropicBetaToResponsesWire(&m, "m", "m").ToMap())
	hasReasoning := false
	for _, item := range gjson.Get(w, "output").Array() {
		if item.Get("type").String() == "reasoning" {
			hasReasoning = true
		}
	}
	if !hasReasoning || !strings.Contains(w, "it is Paris") {
		t.Errorf("thinking block dropped: no reasoning item carries it; output %s", gjson.Get(w, "output").Raw)
	}
}

// findings/chat-responses-refusal-dropped.md: an OpenAI model refusing,
// mostly with structured outputs.
func TestGo2LeanRepro_ChatRefusalReachesResponsesClient(t *testing.T) {
	chat := reproBlChatCompletion(t, `{"id":"c","object":"chat.completion","created":1,"model":"gpt-4o","choices":[{"index":0,
	 "message":{"role":"assistant","content":null,"refusal":"I can't help with that."},
	 "finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":7,"total_tokens":17}}`)
	w := reproBlJSON(t, ConvertChatToResponsesWire(chat, "m", "m").ToMap())
	if !strings.Contains(w, "I can't help with that.") {
		t.Errorf("refusal dropped: the client gets status %s and output %s, an empty completed answer",
			gjson.Get(w, "status").Raw, gjson.Get(w, "output").Raw)
	}
}
