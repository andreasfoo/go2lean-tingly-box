package nonstream

// Reproductions checked with the clients' own SDKs as the oracle: what the
// client's SDK decodes from tingly-box's response must be what it would
// decode from the provider's. Each test fails while the bug is there.

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"google.golang.org/genai"
)

func sdkeqGeminiResp(t *testing.T, js string) *genai.GenerateContentResponse {
	t.Helper()
	var r genai.GenerateContentResponse
	if err := json.Unmarshal([]byte(js), &r); err != nil {
		t.Fatal(err)
	}
	return &r
}

// sdkeqAnthropicBody is what an Anthropic client receives: the gateway's
// WriteAnthropicMessage output, decoded by anthropic-sdk-go.
func sdkeqAnthropicBody(t *testing.T, msg any) anthropic.Message {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	WriteAnthropicMessage(c, msg)
	var out anthropic.Message
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("client cannot decode: %v: %s", err, w.Body.String())
	}
	return out
}

// sdkeqChatBody is what a Chat client's openai-go decodes from the response.
func sdkeqChatBody(t *testing.T, v any) openai.ChatCompletion {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out openai.ChatCompletion
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("client cannot decode: %v: %s", err, raw)
	}
	return out
}

func sdkeqObject(raw string) bool {
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return false
	}
	_, ok := v.(map[string]any)
	return ok
}

// sdkeqExactNumber decodes a JSON object with UseNumber and returns field k verbatim.
func sdkeqExactNumber(t *testing.T, raw []byte, k string) string {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var m map[string]any
	if err := d.Decode(&m); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	n, _ := m[k].(json.Number)
	return n.String()
}

// ---- Finding 62, non-stream trigger: tool call answered with end_turn ----------

func TestGo2LeanRepro_GeminiToAnthropicNonStreamToolCallIsToolUse(t *testing.T) {
	// Gemini ends a function-call answer with finishReason STOP.
	g := sdkeqGeminiResp(t, `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"call_1","name":"get_weather","args":{"city":"Paris"}}}]},"finishReason":"STOP"}]}`)
	for name, resp := range map[string]any{"v1": HandleGoogleToAnthropic(g, "gemini-2.5-pro"), "beta": HandleGoogleToAnthropicBeta(g, "gemini-2.5-pro")} {
		msg := sdkeqAnthropicBody(t, resp)
		var types []string
		for _, b := range msg.Content {
			types = append(types, b.Type)
		}
		if msg.StopReason != anthropic.StopReasonToolUse {
			t.Errorf("%s: content %v with stop_reason %q; want %q", name, types, msg.StopReason, anthropic.StopReasonToolUse)
		}
	}
}

// ---- Finding 66: tool-argument numbers rounded, non-stream -------------------------

func TestGo2LeanRepro_ChatToAnthropicToolArgsExactNonStream(t *testing.T) {
	var chat openai.ChatCompletion
	if err := json.Unmarshal([]byte(`{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"gpt-4o","choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_message","arguments":"{\"id\":9007199254740993}"}}]}}],"usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}}`), &chat); err != nil {
		t.Fatal(err)
	}
	want := sdkeqExactNumber(t, []byte(chat.Choices[0].Message.ToolCalls[0].Function.Arguments), "id")
	beta, err := ConvertOpenAIChatToAnthropicBeta(&chat, "m")
	if err != nil {
		t.Fatal(err)
	}
	for name, msg := range map[string]any{"v1": HandleOpenAIChatToAnthropic(&chat, "m"), "beta": beta} {
		for _, b := range sdkeqAnthropicBody(t, msg).Content {
			if b.Type == "tool_use" {
				if got := sdkeqExactNumber(t, b.Input, "id"); got != want {
					t.Errorf("%s: tool_use input id %s; upstream arguments id %s", name, got, want)
				}
			}
		}
	}
}

func TestGo2LeanRepro_ResponsesToAnthropicToolArgsExactNonStream(t *testing.T) {
	var rs responses.Response
	if err := json.Unmarshal([]byte(`{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"gpt-5","output":[{"id":"fc_1","type":"function_call","call_id":"call_1","name":"get_message","arguments":"{\"id\":9007199254740993}","status":"completed"}],"usage":{"input_tokens":5,"output_tokens":5,"total_tokens":10}}`), &rs); err != nil {
		t.Fatal(err)
	}
	msg := HandleResponsesToAnthropicBeta(&rs, "m")
	for _, b := range sdkeqAnthropicBody(t, &msg).Content {
		if b.Type == "tool_use" {
			if got := sdkeqExactNumber(t, b.Input, "id"); got != "9007199254740993" {
				t.Errorf("tool_use input id %s; upstream arguments id 9007199254740993", got)
			}
		}
	}
}

// ---- Finding 67: argument-less Gemini call, non-stream ------------------------------

func TestGo2LeanRepro_GeminiArgumentlessCallInputIsObjectNonStream(t *testing.T) {
	g := sdkeqGeminiResp(t, `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"call_1","name":"get_time"}}]},"finishReason":"STOP"}]}`)
	for name, msg := range map[string]any{"v1": HandleGoogleToAnthropic(g, "gemini-2.5-pro"), "beta": HandleGoogleToAnthropicBeta(g, "gemini-2.5-pro")} {
		// The client resends the turn: Message.ToParam, as the SDK's tool loop does.
		raw, _ := json.Marshal(sdkeqAnthropicBody(t, msg).ToParam())
		var p struct {
			Content []struct {
				Type  string          `json:"type"`
				Input json.RawMessage `json:"input"`
			} `json:"content"`
		}
		_ = json.Unmarshal(raw, &p)
		for _, b := range p.Content {
			if b.Type == "tool_use" && !sdkeqObject(string(b.Input)) {
				t.Errorf("%s: resent tool_use input %s; want an object such as {} (Anthropic rejects the next request)", name, b.Input)
			}
		}
	}
	for _, tc := range sdkeqChatBody(t, HandleGoogleToOpenAI(g, "gemini-2.5-pro")).Choices[0].Message.ToolCalls {
		if !sdkeqObject(tc.Function.Arguments) {
			t.Errorf("chat: tool call arguments %q; want a JSON object such as {}", tc.Function.Arguments)
		}
	}
}

// ---- Finding 73: Gemini thought text shown as the answer, non-stream ----------------

func TestGo2LeanRepro_GeminiThoughtPartsNotVisibleTextNonStream(t *testing.T) {
	g := sdkeqGeminiResp(t, `{"candidates":[{"content":{"role":"model","parts":[`+
		`{"text":"**Planning** The user wants a greeting.","thought":true},{"text":"Hello!"}]},"finishReason":"STOP"}]}`)
	want := g.Text() // go-genai leaves thought parts out
	if got := sdkeqChatBody(t, HandleGoogleToOpenAI(g, "gemini-2.5-pro")).Choices[0].Message.Content; got != want {
		t.Errorf("chat: content %q; go-genai Text() %q (thought text shown as the answer)", got, want)
	}
	for name, msg := range map[string]any{"v1": HandleGoogleToAnthropic(g, "gemini-2.5-pro"), "beta": HandleGoogleToAnthropicBeta(g, "gemini-2.5-pro")} {
		var text string
		for _, b := range sdkeqAnthropicBody(t, msg).Content {
			if b.Type == "text" {
				text += b.Text
			}
		}
		if text != want {
			t.Errorf("%s: text %q; go-genai Text() %q (thought text shown as the answer)", name, text, want)
		}
	}
}
