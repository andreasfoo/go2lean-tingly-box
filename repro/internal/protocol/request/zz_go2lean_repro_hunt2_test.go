package request

// Reproductions on the request conversions, found by hand (code reading,
// round 2). Each test asserts the behaviour a client expects: it fails while
// the bug is there.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func reproH2ResponsesParams(t *testing.T, raw string) *responses.ResponseNewParams {
	t.Helper()
	var p responses.ResponseNewParams
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

func reproH2ChatParams(t *testing.T, raw string) *openai.ChatCompletionNewParams {
	t.Helper()
	var p openai.ChatCompletionNewParams
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

// findings/responses-developer-message-as-user.md: Codex sends its
// permission instructions as developer messages with input_text content.
func TestGo2LeanRepro_ResponsesDeveloperMessageStaysDeveloper(t *testing.T) {
	p := reproH2ResponsesParams(t, `{"model":"m","input":[
	  {"type":"message","role":"developer","content":[{"type":"input_text","text":"never run rm -rf"}]},
	  {"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}]}`)
	out := ConvertOpenAIResponsesToChat(p, 0)
	if len(out.Messages) == 0 {
		t.Fatal("no messages")
	}
	if m := out.Messages[0]; m.OfDeveloper == nil && m.OfSystem == nil {
		b, _ := json.Marshal(m)
		t.Errorf("developer instruction becomes %s, want a developer (or system) message", b)
	}
}

// findings/chat-anthropic-request-coverage.md: developer message,
// parameterless tool, tool_choice "required" and "none".
func TestGo2LeanRepro_AnthropicRequestKeepsDeveloperAndTools(t *testing.T) {
	p := reproH2ChatParams(t, `{"model":"claude-sonnet-4-5",
	  "messages":[{"role":"developer","content":"Answer in French."},{"role":"user","content":"What time is it?"}],
	  "tools":[{"type":"function","function":{"name":"get_time","description":"current time"}},
	           {"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}],
	  "tool_choice":"required"}`)
	out := ConvertOpenAIToAnthropicRequest(p, 4096)
	if b, _ := json.Marshal(out); !strings.Contains(string(b), "Answer in French.") {
		t.Errorf("developer message dropped (neither in system nor in messages)")
	}
	if len(out.Tools) != 2 {
		t.Errorf("%d of 2 tools converted; the tool without parameters is dropped", len(out.Tools))
	}
	if out.ToolChoice.OfAny == nil {
		b, _ := json.Marshal(out.ToolChoice)
		t.Errorf("tool_choice \"required\" becomes %s, want {\"type\":\"any\"}", b)
	}
	p2 := reproH2ChatParams(t, `{"model":"c","messages":[{"role":"user","content":"hi"}],
	  "tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}],"tool_choice":"none"}`)
	if out2 := ConvertOpenAIToAnthropicRequest(p2, 4096); out2.ToolChoice.OfNone == nil {
		b, _ := json.Marshal(out2.ToolChoice)
		t.Errorf("tool_choice \"none\" becomes %s, want {\"type\":\"none\"}", b)
	}
}

// findings/chat-anthropic-request-coverage.md: temperature, stop and
// parallel_tool_calls.
func TestGo2LeanRepro_AnthropicRequestKeepsSamplingAndStop(t *testing.T) {
	p := reproH2ChatParams(t, `{"model":"c","messages":[{"role":"user","content":"hi"}],
	  "temperature":0,"stop":["STOP"],"parallel_tool_calls":false,
	  "tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}]}`)
	out := ConvertOpenAIToAnthropicRequest(p, 4096)
	if !out.Temperature.Valid() {
		t.Errorf("temperature 0 not carried")
	}
	if len(out.StopSequences) != 1 || out.StopSequences[0] != "STOP" {
		t.Errorf("stop [STOP] not carried as stop_sequences (got %v)", out.StopSequences)
	}
	b, _ := json.Marshal(out.ToolChoice)
	if !strings.Contains(string(b), `"disable_parallel_tool_use":true`) {
		t.Errorf("parallel_tool_calls false not carried: tool_choice %s, want disable_parallel_tool_use true", b)
	}
}

// findings/chat-responses-request-params.md.
func TestGo2LeanRepro_ResponsesRequestKeepsTokensAndSampling(t *testing.T) {
	p := reproH2ChatParams(t, `{"model":"gpt-4.1","messages":[{"role":"user","content":"hi"}],
	  "temperature":0,"top_p":0,"max_completion_tokens":50,"parallel_tool_calls":false,
	  "tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}]}`)
	out := ConvertChatToOpenAIResponses(p, 0)
	if !out.MaxOutputTokens.Valid() || out.MaxOutputTokens.Value != 50 {
		t.Errorf("max_completion_tokens 50 not carried to max_output_tokens (valid=%v value=%v)", out.MaxOutputTokens.Valid(), out.MaxOutputTokens.Value)
	}
	if !out.Temperature.Valid() {
		t.Errorf("temperature 0 dropped")
	}
	if !out.TopP.Valid() {
		t.Errorf("top_p 0 dropped")
	}
	if !out.ParallelToolCalls.Valid() || out.ParallelToolCalls.Value {
		t.Errorf("parallel_tool_calls false dropped")
	}
}
