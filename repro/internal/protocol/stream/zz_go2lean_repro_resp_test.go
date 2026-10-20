package stream

// Reproduction on the Responses -> Chat and Responses -> Anthropic stream
// converters, from bug class 15 of hunting-similar-tools.md. Each test
// asserts the behaviour a client expects: it fails while the bug is there.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"

	"github.com/tingly-dev/tingly-box/internal/protocol/wire"
)

// reproDoneOnlyStream is a Responses stream whose function call arguments
// arrive only in response.function_call_arguments.done and
// response.output_item.done, without delta events (as gpt-5.3-codex-spark
// streams them on the ChatGPT Codex backend, BerriAI/litellm#27144).
func reproDoneOnlyStream(t *testing.T) ResponsesStreamIter {
	t.Helper()
	const args = `{\"city\":\"Paris\"}`
	events := []string{
		`{"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress","output":[]}}`,
		`{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"get_weather","arguments":"","status":"in_progress"}}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"` + args + `"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"get_weather","arguments":"` + args + `","status":"completed"}}`,
		`{"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[{"id":"fc_1","type":"function_call","call_id":"call_1","name":"get_weather","arguments":"` + args + `","status":"completed"}],"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`,
	}
	var b strings.Builder
	for _, e := range events {
		var probe map[string]any
		if err := json.Unmarshal([]byte(e), &probe); err != nil {
			t.Fatalf("bad event %s: %v", e, err)
		}
		b.WriteString("event: " + probe["type"].(string) + "\ndata: " + e + "\n\n")
	}
	resp := &http.Response{Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(b.String()))}
	return ssestream.NewStream[responses.ResponseStreamEventUnion](ssestream.NewDecoder(resp), nil)
}

const reproDoneOnlyArgs = `{"city":"Paris"}`

// findings/responses-done-only-arguments.md, Responses -> Chat.
func TestGo2LeanRepro_DoneOnlyArgumentsReachChatClient(t *testing.T) {
	conv := NewOpenAIResponsesToChatConverter(reproDoneOnlyStream(t), "m", false)
	args := ""
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
					if tc.Function.Arguments != nil {
						args += *tc.Function.Arguments
					}
				}
			}
		}
	}
	if args != reproDoneOnlyArgs {
		t.Errorf("streamed tool call arguments %q, want %q (sent only in the .done events)", args, reproDoneOnlyArgs)
	}
}

// findings/responses-done-only-arguments.md, Responses -> Anthropic.
func TestGo2LeanRepro_DoneOnlyArgumentsReachAnthropicClient(t *testing.T) {
	conv := NewOpenAIResponsesToAnthropicConverter(context.Background(), reproDoneOnlyStream(t), "m")
	partial, startInput := "", ""
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		ae, ok := AsAnthropicEvent(ev)
		if !ok {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(ae.RawJSON()), &m); err != nil {
			t.Fatal(err)
		}
		switch ae.Type {
		case "content_block_start":
			if cb, _ := m["content_block"].(map[string]any); cb["type"] == "tool_use" {
				in, _ := json.Marshal(cb["input"])
				startInput = string(in)
			}
		case "content_block_delta":
			if d, _ := m["delta"].(map[string]any); d["type"] == "input_json_delta" {
				partial += d["partial_json"].(string)
			}
		}
	}
	got := partial
	if got == "" {
		got = startInput // an Anthropic client uses the start input when no deltas follow
	}
	if got != reproDoneOnlyArgs {
		t.Errorf("tool_use input %q, want %q (sent only in the .done events)", got, reproDoneOnlyArgs)
	}
}
