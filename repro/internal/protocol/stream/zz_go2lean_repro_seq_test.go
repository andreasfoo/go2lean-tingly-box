package stream

// Reproductions on the Responses -> Anthropic stream converter, found while
// triaging go2lean:sequence specs (go2lean's M2 agent on this package,
// 2026-10-22). Each test asserts what an Anthropic client expects: it fails
// while the bug is there.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
)

// reproSeqStream is a Responses SSE stream of the given event JSON texts.
func reproSeqStream(t *testing.T, events ...string) ResponsesStreamIter {
	t.Helper()
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

// reproSeqAnthropicEvents runs the Responses -> Anthropic converter over the
// stream and returns the Anthropic events as JSON maps, in order.
func reproSeqAnthropicEvents(t *testing.T, stream ResponsesStreamIter) []map[string]any {
	t.Helper()
	conv := NewOpenAIResponsesToAnthropicConverter(context.Background(), stream, "m")
	var out []map[string]any
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			return out
		}
		ae, ok := AsAnthropicEvent(ev)
		if !ok {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(ae.RawJSON()), &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
}

// findings/responses-anthropic-stream-tool-then-text-end-turn.md: a response
// with a tool call followed by text ends with stop_reason end_turn; the
// client is told the turn is over and never runs the tool.
func TestGo2LeanRepro_ResponsesToAnthropicToolThenTextIsToolUse(t *testing.T) {
	events := reproSeqAnthropicEvents(t, reproSeqStream(t,
		`{"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress","output":[]}}`,
		`{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"get_weather","arguments":"","status":"in_progress"}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{}"}`,
		`{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"get_weather","arguments":"{}","status":"completed"}}`,
		`{"type":"response.output_item.added","output_index":1,"item":{"id":"msg_1","type":"message","role":"assistant","status":"in_progress","content":[]}}`,
		`{"type":"response.content_part.added","item_id":"msg_1","output_index":1,"content_index":0,"part":{"type":"output_text","text":"","annotations":[]}}`,
		`{"type":"response.output_text.delta","item_id":"msg_1","output_index":1,"content_index":0,"delta":"Checking."}`,
		`{"type":"response.content_part.done","item_id":"msg_1","output_index":1,"content_index":0,"part":{"type":"output_text","text":"Checking.","annotations":[]}}`,
		`{"type":"response.output_item.done","output_index":1,"item":{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Checking.","annotations":[]}]}}`,
		`{"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[]}}`,
	))
	for _, e := range events {
		if e["type"] == "message_delta" {
			d, _ := e["delta"].(map[string]any)
			if d["stop_reason"] != "tool_use" {
				t.Errorf("stop_reason %v, want tool_use (the message has a tool_use block)", d["stop_reason"])
			}
			return
		}
	}
	t.Fatal("no message_delta")
}

// findings/responses-anthropic-stream-incomplete-open-tool.md: a response
// cut by max_output_tokens in the middle of a tool call ends with the
// tool_use block still open when message_delta and message_stop arrive.
func TestGo2LeanRepro_ResponsesToAnthropicIncompleteClosesToolBlock(t *testing.T) {
	events := reproSeqAnthropicEvents(t, reproSeqStream(t,
		`{"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress","output":[]}}`,
		`{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"write_file","arguments":"","status":"in_progress"}}`,
		`{"type":"response.function_call_arguments.delta","item_id":"fc_1","output_index":0,"delta":"{\"path\":\"a.txt\",\"content\":\"abc"}`,
		`{"type":"response.incomplete","response":{"id":"resp_1","object":"response","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}}`,
	))
	open := map[float64]bool{}
	for _, e := range events {
		switch e["type"] {
		case "content_block_start":
			open[e["index"].(float64)] = true
		case "content_block_stop":
			delete(open, e["index"].(float64))
		case "message_delta":
			if len(open) > 0 {
				t.Errorf("message_delta with content blocks still open: %v", open)
			}
			return
		}
	}
	t.Fatal("no message_delta")
}

// findings/responses-anthropic-stream-summary-part-text.md: summary text
// carried by response.reasoning_summary_part.added is dropped (the branch
// compares the part type with "text"; the Responses API sends
// "summary_text"). Latent: OpenAI sends that part empty and the text in
// reasoning_summary_text.delta events, which are handled.
func TestGo2LeanRepro_ResponsesToAnthropicSummaryPartText(t *testing.T) {
	events := reproSeqAnthropicEvents(t, reproSeqStream(t,
		`{"type":"response.created","response":{"id":"resp_1","object":"response","status":"in_progress","output":[]}}`,
		`{"type":"response.output_item.added","output_index":0,"item":{"id":"rs_1","type":"reasoning","summary":[]}}`,
		`{"type":"response.reasoning_summary_part.added","item_id":"rs_1","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":"Plan the call."}}`,
		`{"type":"response.reasoning_summary_part.done","item_id":"rs_1","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":"Plan the call."}}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"Plan the call."}]}}`,
		`{"type":"response.completed","response":{"id":"resp_1","object":"response","status":"completed","output":[]}}`,
	))
	thinking := ""
	for _, e := range events {
		if e["type"] == "content_block_delta" {
			if d, _ := e["delta"].(map[string]any); d["type"] == "thinking_delta" {
				thinking += d["thinking"].(string)
			}
		}
	}
	if thinking != "Plan the call." {
		t.Errorf("thinking %q, want the summary text %q", thinking, "Plan the call.")
	}
}
