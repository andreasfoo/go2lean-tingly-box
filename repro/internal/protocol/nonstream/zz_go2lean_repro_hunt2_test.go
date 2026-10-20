package nonstream

// Reproductions on the non-streaming converters, found by hand (code
// reading, round 2). Each test asserts the behaviour a client expects: it
// fails while the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
)

// findings/anthropic-abnormal-stop-reasons.md: Claude Sonnet 4.5+ ends a
// turn that ran into the context window with stop_reason
// "model_context_window_exceeded"; a streaming-classifier intervention
// ends with "refusal". Neither is a normal completion.
func TestGo2LeanRepro_AbnormalStopReasonsNotCompleted(t *testing.T) {
	for _, sr := range []string{"model_context_window_exceeded", "refusal"} {
		var m anthropic.BetaMessage
		if err := json.Unmarshal([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"partial answer"}],"stop_reason":"`+sr+`","stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":5}}`), &m); err != nil {
			t.Fatal(err)
		}
		chat := ConvertAnthropicBetaToOpenAIChat(&m, "claude")
		if fr := chat.Choices[0].FinishReason; fr == "stop" {
			t.Errorf("stop_reason %q becomes Chat finish_reason %q, the normal end of a turn", sr, fr)
		}
		resp := ConvertAnthropicBetaToResponsesWire(&m, "claude", "claude")
		if resp.Status == "completed" {
			t.Errorf("stop_reason %q becomes Responses status %q, the normal end of a turn", sr, resp.Status)
		}
	}
}

// findings/chat-tool-call-empty-id.md, non-stream: an OpenAI-compatible
// upstream (Gemini's compat layer) returns a tool call without "id".
func TestGo2LeanRepro_NonStreamToolCallWithoutIDGetsID(t *testing.T) {
	var chat openai.ChatCompletion
	if err := json.Unmarshal([]byte(`{"id":"x","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":"tool_calls"}]}`), &chat); err != nil {
		t.Fatal(err)
	}
	resp := ConvertChatToResponsesWire(&chat, "m", "m")
	if resp.Output[0].CallID == "" {
		t.Errorf("Chat -> Responses: function_call has call_id \"\" (dropped from the JSON by omitempty), want a generated call_id")
	}
	msg, err := ConvertOpenAIChatToAnthropicBeta(&chat, "m")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content[0].ID == "" {
		t.Errorf("Chat -> Anthropic: tool_use has id \"\", want a generated id")
	}
}
