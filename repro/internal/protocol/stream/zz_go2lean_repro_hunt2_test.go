package stream

// Reproductions on the stream converters, found by hand (code reading,
// round 2). Each test asserts the behaviour a client expects: it fails while
// the bug is there. The stream helpers (openAIChatChunks,
// openAIChatSliceStream, newResponsesIterFromEvents, anthropicBetaEvents,
// anthropicBetaSliceStream) are tingly-box's own, from the package's tests.

import (
	"encoding/json"
	"testing"

	"github.com/tingly-dev/tingly-box/internal/protocol/wire"
)

// findings/responses-stream-trailing-usage.md: with
// stream_options.include_usage an OpenAI Chat stream sends usage in a last
// chunk with "choices": [] after the finish_reason chunk.
func TestGo2LeanRepro_ResponsesStreamUsageAfterFinishChunk(t *testing.T) {
	chunks := openAIChatChunks(t,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[],"usage":{"prompt_tokens":1200,"completion_tokens":7,"total_tokens":1207,"prompt_tokens_details":{"cached_tokens":1024}}}`,
	)
	conv := NewChatToResponsesConverter(&openAIChatSliceStream{chunks: chunks}, "gpt-4o")
	var completed *wire.ResponsesCompletedEvent
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		if c, ok := ev.(wire.ResponsesCompletedEvent); ok {
			completed = &c
		}
	}
	if completed == nil {
		t.Fatal("no response.completed")
	}
	u := completed.Response.Usage
	if u == nil || u.InputTokens != 1200 || u.OutputTokens != 7 {
		b, _ := json.Marshal(u)
		t.Errorf("response.completed carries usage %s, want input_tokens 1200, output_tokens 7 (the upstream usage chunk after finish_reason)", b)
	}
}

// findings/responses-chat-tool-call-index.md: a reasoning model's first
// function_call is at output_index 1 (the reasoning item is at 0).
func TestGo2LeanRepro_ToolCallIndexAfterReasoningItem(t *testing.T) {
	events := []map[string]any{
		{"type": "response.created", "response": map[string]any{"id": "resp_1", "object": "response", "status": "in_progress", "output": []any{}}},
		{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{}}},
		{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{}}},
		{"type": "response.output_item.added", "output_index": 1, "item": map[string]any{"id": "fc_1", "type": "function_call", "call_id": "call_1", "name": "get_weather", "arguments": "", "status": "in_progress"}},
		{"type": "response.function_call_arguments.delta", "item_id": "fc_1", "output_index": 1, "delta": `{"city":"Paris"}`},
		{"type": "response.function_call_arguments.done", "item_id": "fc_1", "output_index": 1, "arguments": `{"city":"Paris"}`},
		{"type": "response.completed", "response": map[string]any{"id": "resp_1", "object": "response", "status": "completed",
			"output": []any{
				map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{}},
				map[string]any{"id": "fc_1", "type": "function_call", "call_id": "call_1", "name": "get_weather", "arguments": `{"city":"Paris"}`},
			},
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}}},
	}
	conv := NewOpenAIResponsesToChatConverter(newResponsesIterFromEvents(events), "gpt-5", false)
	seen := map[int]bool{}
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		if c, ok := ev.(wire.ChatStreamChunk); ok {
			for _, ch := range c.Choices {
				for _, tc := range ch.Delta.ToolCalls {
					seen[tc.Index] = true
				}
			}
		}
	}
	for i := range seen {
		if i != 0 {
			t.Errorf("the only tool call is streamed with tool_calls[].index %d, want 0 (Chat indexes count tool calls, not Responses output items)", i)
		}
	}
}

// findings/anthropic-abnormal-stop-reasons.md, Anthropic -> Responses stream.
func TestGo2LeanRepro_ContextWindowExceededStreamIncomplete(t *testing.T) {
	events := anthropicBetaEvents(t,
		map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-4-5", "content": []any{}, "stop_reason": nil, "usage": map[string]any{"input_tokens": 190000, "output_tokens": 1}}},
		map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}},
		map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": "partial"}},
		map[string]any{"type": "content_block_stop", "index": 0},
		map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "model_context_window_exceeded", "stop_sequence": nil}, "usage": map[string]any{"output_tokens": 9000}},
		map[string]any{"type": "message_stop"},
	)
	conv := NewAnthropicBetaToOpenAIResponsesConverter(&anthropicBetaSliceStream{events: events}, "claude")
	last := ""
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		b, _ := json.Marshal(ev)
		last = string(b)
	}
	var term struct {
		Type     string `json:"type"`
		Response struct {
			Status string `json:"status"`
		} `json:"response"`
	}
	_ = json.Unmarshal([]byte(last), &term)
	if term.Type == "response.completed" || term.Response.Status == "completed" {
		t.Errorf("stop_reason model_context_window_exceeded streamed as %s with status %q, want an incomplete response (as max_tokens is)", term.Type, term.Response.Status)
	}
}

// findings/chat-tool-call-empty-id.md, Chat -> Anthropic stream: an
// OpenAI-compatible upstream (Gemini's compat layer) sends a tool call
// without "id".
func TestGo2LeanRepro_StreamToolCallWithoutIDGetsID(t *testing.T) {
	chunks := openAIChatChunks(t,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":null}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
	)
	conv := NewOpenAIChatToAnthropicBetaConverter(&openAIChatSliceStream{chunks: chunks}, "m", nil)
	starts := 0
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		a, ok := AsAnthropicEvent(ev)
		if !ok || a.Type != "content_block_start" {
			continue
		}
		var e struct {
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"content_block"`
		}
		_ = json.Unmarshal([]byte(a.RawJSON()), &e)
		if e.ContentBlock.Type != "tool_use" {
			continue
		}
		starts++
		if e.ContentBlock.ID == "" {
			t.Errorf("tool_use streamed with id \"\", want a generated id (a tool_result cannot refer to it)")
		}
	}
	if starts == 0 {
		t.Fatal("no tool_use block streamed")
	}
}
