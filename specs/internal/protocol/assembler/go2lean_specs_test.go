package assembler

import (
	"encoding/json"
	"strconv"

	"github.com/openai/openai-go/v3/responses"
)

// Hand-written go2lean specs for the stream assemblers (findings 37, 41,
// 42, round 4).

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// go2lean:spec ResponsesAssembler.Accumulate
// SpecResponsesAssemblerAccumulateKeepsItems: assembling the stream of the
// event's response (each output item added and done, text as deltas, then
// response.completed carrying the output or, as the ChatGPT Codex backend
// sends it, an empty output: the sequence number picks) gives a response
// that holds every function call of the stream.
func SpecResponsesAssemblerAccumulateKeepsItems(a *ResponsesAssembler, event responses.ResponseStreamEventUnion, ret bool) bool {
	r := event.Response
	asm := NewResponsesAssemblerWithID("resp_go2lean", "msg_go2lean")
	feed := func(text string) {
		var ev responses.ResponseStreamEventUnion
		if json.Unmarshal([]byte(text), &ev) == nil {
			asm.Accumulate(ev)
		}
	}
	var calls []string
	feed(`{"type":"response.created","response":{"id":"resp_go2lean","status":"in_progress","output":[]}}`)
	for i, item := range r.Output {
		raw := item.RawJSON()
		if raw == "" {
			return true
		}
		at := `"output_index":` + strconv.Itoa(i) + `,"item_id":` + jsonQuote(item.ID)
		feed(`{"type":"response.output_item.added",` + at + `,"item":` + raw + `}`)
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" {
					feed(`{"type":"response.output_text.delta",` + at + `,"delta":` + jsonQuote(part.Text) + `}`)
				}
			}
		case "function_call":
			calls = append(calls, item.CallID)
			feed(`{"type":"response.function_call_arguments.delta",` + at + `,"delta":` + jsonQuote(item.Arguments.OfString) + `}`)
		default:
			return true
		}
		feed(`{"type":"response.output_item.done",` + at + `,"item":` + raw + `}`)
	}
	output := "[]"
	if event.SequenceNumber%2 == 1 && r.RawJSON() != "" {
		output = ""
	}
	if output == "" {
		feed(`{"type":"response.completed","response":` + r.RawJSON() + `}`)
	} else {
		feed(`{"type":"response.completed","response":{"id":"resp_go2lean","status":"completed","output":[]}}`)
	}
	out := asm.Finish()
	if out == nil {
		return len(calls) == 0
	}
	for _, id := range calls {
		found := false
		for _, item := range out.Output {
			if item.Type == "function_call" && item.CallID == id {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
