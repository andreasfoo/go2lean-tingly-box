package stream

// Reproductions of the bugs found by go2lean from the issue-derived
// specs (run "issues1"). Each test fails while the bug is there.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3/responses"

	"github.com/tingly-dev/tingly-box/internal/protocol/nonstream"
)

// ---- Finding 81: content_filter is end_turn on the stream, refusal without (#1132 class) ----

// An Azure OpenAI / OpenAI Responses upstream stops on its content filter
// (response.incomplete, incomplete_details.reason "content_filter"). The
// non-stream Responses -> Anthropic path reports stop_reason "refusal"; the
// stream path reports "end_turn", so a filtered, cut-short answer looks
// like a normal completion to a streaming Anthropic client.
func TestGo2LeanRepro_ResponsesToAnthropicStreamContentFilterAgrees(t *testing.T) {
	raw := `{"type":"response.incomplete","sequence_number":3,"response":{"id":"resp_1","object":"response","created_at":1,"status":"incomplete","incomplete_details":{"reason":"content_filter"},"model":"gpt-5","output":[{"id":"msg_1","type":"message","status":"incomplete","role":"assistant","content":[{"type":"output_text","text":"Here is how","annotations":[]}]}],"usage":{"input_tokens":12,"input_tokens_details":{"cached_tokens":0},"output_tokens":3,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":15}}}`
	var ev responses.ResponseStreamEventUnion
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	resp := ev.AsResponseIncomplete().Response
	want := string(nonstream.HandleResponsesToAnthropicBeta(&resp, "m").StopReason)

	conv := newResponsesToAnthropicConverter(context.Background(), nil, "m")
	conv.processEvent(ev)
	got := ""
	for _, p := range conv.pending {
		e, ok := p.(anthropicStreamEvent)
		if !ok || e.eventType != eventTypeMessageDelta {
			continue
		}
		if m, ok := e.data.(map[string]interface{}); ok {
			if d, ok := m["delta"].(map[string]interface{}); ok {
				got, _ = d["stop_reason"].(string)
			}
		}
	}
	if got != want {
		t.Errorf("stream stop_reason %q; non-stream stop_reason for the same response %q", got, want)
	}
}
