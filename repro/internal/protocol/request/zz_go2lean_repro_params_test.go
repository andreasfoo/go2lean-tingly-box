package request

// Reproduction on the Anthropic -> Chat request conversion, from bug classes
// 26 and 29 of hunting-similar-tools.md. Each test asserts the behaviour a
// client expects: it fails while the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func reproBetaRequest(t *testing.T, raw string) *anthropic.BetaMessageNewParams {
	t.Helper()
	var req anthropic.BetaMessageNewParams
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

// findings/chat-request-params.md: temperature, top_p and stop_sequences
// reach a Responses provider (temperature, top_p) but not a Chat provider.
func TestGo2LeanRepro_ChatRequestKeepsSamplingAndStop(t *testing.T) {
	req := reproBetaRequest(t, `{"model":"m","max_tokens":100,"temperature":0,"top_p":0.5,
		"stop_sequences":["STOP"],"messages":[{"role":"user","content":"hi"}]}`)
	out, _ := ConvertAnthropicBetaToOpenAIRequest(req, false, false, false)
	if !out.Temperature.Valid() || out.Temperature.Value != 0 {
		t.Errorf("temperature 0 not carried (valid=%v value=%v)", out.Temperature.Valid(), out.Temperature.Value)
	}
	if !out.TopP.Valid() || out.TopP.Value != 0.5 {
		t.Errorf("top_p 0.5 not carried (valid=%v value=%v)", out.TopP.Valid(), out.TopP.Value)
	}
	if stop := out.Stop.OfStringArray; len(stop) != 1 || stop[0] != "STOP" {
		if out.Stop.OfString.Value != "STOP" {
			t.Errorf("stop_sequences [STOP] not carried as stop")
		}
	}
}

// findings/chat-request-params.md: tool_choice "any" (must call a tool) and
// "none" (must not) both become "auto" on a Chat provider; the Responses
// conversion maps "any" to "required".
func TestGo2LeanRepro_ChatRequestKeepsToolChoiceMode(t *testing.T) {
	for choice, want := range map[string]string{"any": "required", "none": "none"} {
		req := reproBetaRequest(t, `{"model":"m","max_tokens":100,"messages":[{"role":"user","content":"hi"}],
			"tools":[{"name":"f","input_schema":{"type":"object"}}],"tool_choice":{"type":"`+choice+`"}}`)
		out, _ := ConvertAnthropicBetaToOpenAIRequest(req, false, false, false)
		if got := out.ToolChoice.OfAuto.Value; got != want {
			t.Errorf("tool_choice %q becomes %q, want %q", choice, got, want)
		}
	}
}
