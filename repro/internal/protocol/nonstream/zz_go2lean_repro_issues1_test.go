package nonstream

// Reproductions of the bugs found by go2lean from the issue-derived
// specs (run "issues1"). Each test fails while the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"google.golang.org/genai"
)

func issues1Parse(t *testing.T, body string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("parse %s: %v", body, err)
	}
}

// ---- Finding 65, further trigger: tool_use input null in a response (#1840 class) ----

// An OpenAI-compatible upstream (vLLM-style tool parsers json.dumps a
// missing arguments value) answers a parameterless call with
// arguments "null". The Anthropic client gets "input": null; its SDK and
// tool validation expect an object, and replaying the turn to Anthropic is
// rejected ("tool_use.input: Input should be a valid dictionary").
func TestGo2LeanRepro_ChatToAnthropicNullArgumentsInputIsObject(t *testing.T) {
	var chat openai.ChatCompletion
	issues1Parse(t, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"qwen3","choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_time","arguments":"null"}}]}}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`, &chat)
	beta, err := ConvertOpenAIChatToAnthropicBeta(&chat, "m")
	if err != nil {
		t.Fatal(err)
	}
	v1 := HandleOpenAIChatToAnthropic(&chat, "m")
	for name, content := range map[string]any{"beta": beta.Content, "v1": v1.Content} {
		raw, _ := json.Marshal(content)
		var blocks []struct {
			Type  string          `json:"type"`
			Input json.RawMessage `json:"input"`
		}
		_ = json.Unmarshal(raw, &blocks)
		for _, b := range blocks {
			if b.Type != "tool_use" {
				continue
			}
			var v any
			_ = json.Unmarshal(b.Input, &v)
			if _, ok := v.(map[string]any); !ok {
				t.Errorf("%s: tool_use input %s; Anthropic clients expect an object ({})", name, b.Input)
			}
		}
	}
}

// ---- Finding 70, further trigger: Responses -> Chat non-stream drops reasoning ----

func TestGo2LeanRepro_ResponsesToChatNonStreamKeepsReasoning(t *testing.T) {
	var rs responses.Response
	issues1Parse(t, `{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"gpt-5","output":[`+
		`{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"**Greeting** The user says hi."}]},`+
		`{"id":"msg_1","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"Hello!","annotations":[]}]}],`+
		`"usage":{"input_tokens":5,"output_tokens":20,"total_tokens":25,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":12}}}`, &rs)
	got := ConvertResponsesToOpenAIChat(&rs, "m")
	if r := got.Choices[0].Message.ReasoningContent; r != "**Greeting** The user says hi." {
		t.Errorf("chat client reasoning_content %q; upstream reasoning summary %q", r, "**Greeting** The user says hi.")
	}
}

// ---- Finding 80: Gemini -> Chat drops cached prompt tokens (#1181 class) ----

func TestGo2LeanRepro_GeminiToChatKeepsCachedTokens(t *testing.T) {
	var resp genai.GenerateContentResponse
	issues1Parse(t, `{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello!"}]},"finishReason":"STOP","index":0}],`+
		`"usageMetadata":{"promptTokenCount":1000,"cachedContentTokenCount":900,"candidatesTokenCount":50,"totalTokenCount":1050},"modelVersion":"gemini-2.5-flash"}`, &resp)
	out := HandleGoogleToOpenAI(&resp, "m")
	raw, _ := json.Marshal(out["usage"])
	var usage struct {
		PromptTokensDetails struct {
			CachedTokens int64 `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	}
	_ = json.Unmarshal(raw, &usage)
	if usage.PromptTokensDetails.CachedTokens != 900 {
		t.Errorf("chat client usage %s; want prompt_tokens_details.cached_tokens 900 (Gemini cachedContentTokenCount)", raw)
	}
}
