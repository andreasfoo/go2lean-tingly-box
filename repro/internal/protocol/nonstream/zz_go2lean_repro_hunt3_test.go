package nonstream

// Reproductions on the non-streaming converters, found by hand (code
// reading, round 3). Each test asserts the behaviour a client expects: it
// fails while the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3/responses"
	"github.com/tidwall/gjson"
)

func reproH3Response(t *testing.T, body string) *responses.Response {
	t.Helper()
	var rs responses.Response
	if err := json.Unmarshal([]byte(body), &rs); err != nil {
		t.Fatal(err)
	}
	return &rs
}

// findings/responses-anthropic-nonstream-refusal.md: the refusal text is in
// content[].refusal of a "refusal" part.
func TestGo2LeanRepro_ResponsesRefusalReachesAnthropicClient(t *testing.T) {
	rs := reproH3Response(t, `{"id":"resp_1","object":"response","status":"completed","model":"gpt-4o",
	 "output":[{"id":"msg_1","type":"message","role":"assistant","status":"completed",
	   "content":[{"type":"refusal","refusal":"I can't help with that."}]}],
	 "usage":{"input_tokens":10,"output_tokens":7,"total_tokens":17}}`)
	msg := HandleResponsesToAnthropicBeta(rs, "m")
	raw := msg.RawJSON()
	found := false
	for _, b := range msg.Content {
		if b.Type == "text" && b.Text == "I can't help with that." {
			found = true
		}
	}
	if !gjson.Get(raw, "content").IsArray() || !found {
		t.Errorf("refusal text lost (want a text block with it, content an array); client receives content %s, stop_reason %s",
			gjson.Get(raw, "content").Raw, gjson.Get(raw, "stop_reason").Raw)
	}
}

// findings/responses-anthropic-nonstream-reasoning-summary.md: OpenAI
// reasoning models return reasoning as summary[].summary_text.
func TestGo2LeanRepro_NonStreamReasoningSummaryIsThinking(t *testing.T) {
	rs := reproH3Response(t, `{"id":"resp_1","object":"response","status":"completed","model":"o4-mini",
	 "output":[
	  {"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"Compared both options."}]},
	  {"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Option A.","annotations":[]}]}],
	 "usage":{"input_tokens":10,"output_tokens":7,"total_tokens":17}}`)
	msg := HandleResponsesToAnthropicBeta(rs, "m")
	if len(msg.Content) == 0 || msg.Content[0].Type != "thinking" || msg.Content[0].Thinking != "Compared both options." {
		t.Errorf("reasoning summary not carried as the leading thinking block; content = %s", gjson.Get(msg.RawJSON(), "content").Raw)
	}
}
