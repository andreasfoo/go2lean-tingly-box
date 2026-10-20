package assembler

// Reproductions on the stream assemblers (Responses: the Codex non-stream
// path; Anthropic: recordings), found by hand (code reading, round 4). Each
// test asserts the behaviour a caller expects: it fails while the bug is
// there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3/responses"
)

func reproH4Feed(t *testing.T, a *ResponsesAssembler, events []string) {
	t.Helper()
	for _, raw := range events {
		var e responses.ResponseStreamEventUnion
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			t.Fatal(err)
		}
		a.Accumulate(e)
	}
}

// A ChatGPT Codex stream: the items arrive only as output_item events and
// response.completed carries "output": [] (stage/upstream/openai.go
// completeFromStream handles the same backend).
var reproH4CodexToolStream = []string{
	`{"type":"response.created","sequence_number":0,"response":{"id":"resp_1","object":"response","status":"in_progress","output":[]}}`,
	`{"type":"response.output_item.added","sequence_number":1,"output_index":0,"item":{"id":"msg_1","type":"message","role":"assistant","status":"in_progress","content":[]}}`,
	`{"type":"response.output_text.delta","sequence_number":2,"item_id":"msg_1","output_index":0,"content_index":0,"delta":"Let me check."}`,
	`{"type":"response.output_item.done","sequence_number":3,"output_index":0,"item":{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Let me check.","annotations":[]}]}}`,
	`{"type":"response.output_item.added","sequence_number":4,"output_index":1,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"shell","arguments":"","status":"in_progress"}}`,
	`{"type":"response.function_call_arguments.delta","sequence_number":5,"item_id":"fc_1","output_index":1,"delta":"{\"cmd\":\"ls\"}"}`,
	`{"type":"response.function_call_arguments.done","sequence_number":6,"item_id":"fc_1","output_index":1,"arguments":"{\"cmd\":\"ls\"}"}`,
	`{"type":"response.output_item.done","sequence_number":7,"output_index":1,"item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"shell","arguments":"{\"cmd\":\"ls\"}","status":"completed"}}`,
	`{"type":"response.completed","sequence_number":8,"response":{"id":"resp_1","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`,
}

// findings/codex-nonstream-function-calls-dropped.md: Finish fills an empty
// terminal output with the streamed text only; the function call is lost.
func TestGo2LeanRepro_CodexNonStreamKeepsFunctionCall(t *testing.T) {
	a := NewResponsesAssembler()
	reproH4Feed(t, a, reproH4CodexToolStream)
	resp := a.Finish()
	var types []string
	for _, it := range resp.Output {
		types = append(types, it.Type)
	}
	if len(types) != 2 || types[1] != "function_call" {
		t.Errorf("assembled output item types %v: the streamed function_call is lost; want [message function_call]", types)
	}
}

// findings/responses-assembler-tool-call-arguments.md: ToolCalls() reads
// event.Arguments on response.function_call_arguments.delta, whose wire
// field is "delta", so the arguments are always empty.
func TestGo2LeanRepro_ResponsesAssemblerToolCallArguments(t *testing.T) {
	a := NewResponsesAssembler()
	reproH4Feed(t, a, reproH4CodexToolStream)
	tc := a.ToolCalls()[1]
	if tc.Arguments != `{"cmd":"ls"}` {
		t.Errorf("ToolCalls()[1].Arguments = %q after the delta %q; want %q", tc.Arguments, `{"cmd":"ls"}`, `{"cmd":"ls"}`)
	}
}

// findings/recording-anthropic-stream-usage.md: the recording assembler
// ignores message_start usage and replaces usage from message_delta, so the
// input and cache counts are lost.
func TestGo2LeanRepro_RecordingAssemblerKeepsInputUsage(t *testing.T) {
	a := NewAnthropicStreamAssembler()
	for _, raw := range []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":null,"usage":{"input_tokens":1200,"cache_creation_input_tokens":300,"cache_read_input_tokens":5000,"output_tokens":1}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":42}}`,
		`{"type":"message_stop"}`,
	} {
		var e anthropic.MessageStreamEventUnion
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			t.Fatal(err)
		}
		a.RecordV1Event(&e)
	}
	u := a.Finish("claude-sonnet-4-5", 0, 0).Usage
	if u.InputTokens != 1200 || u.CacheCreationInputTokens != 300 || u.CacheReadInputTokens != 5000 || u.OutputTokens != 42 {
		t.Errorf("assembled usage in=%d cache_write=%d cache_read=%d out=%d; want in=1200 cache_write=300 cache_read=5000 (from message_start) out=42",
			u.InputTokens, u.CacheCreationInputTokens, u.CacheReadInputTokens, u.OutputTokens)
	}
}
