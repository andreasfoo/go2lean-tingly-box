package stream

// Reproductions on the stream converters, found by hand (code reading,
// round 3). Each test asserts the behaviour a client expects: it fails while
// the bug is there. The stream helpers (anthropicBetaEvents,
// anthropicBetaSliceStream, newFakeResponsesDecoder, eventsToStrings) are
// tingly-box's own, from the package's tests.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	openaistream "github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
	"github.com/tidwall/gjson"

	"github.com/tingly-dev/tingly-box/internal/protocol/wire"
)

// findings/parameterless-tool-stream-arguments.md: a parameterless tool call
// as the Anthropic API streams it: input {} in content_block_start and one
// input_json_delta with an empty partial_json.
func reproH3ParameterlessToolStream(t *testing.T) *anthropicBetaSliceStream {
	return &anthropicBetaSliceStream{events: anthropicBetaEvents(t,
		map[string]any{"type": "message_start", "message": map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "content": []any{}, "model": "claude",
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 1}}},
		map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{
			"type": "tool_use", "id": "toolu_1", "name": "get_time", "input": map[string]any{}}},
		map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": ""}},
		map[string]any{"type": "content_block_stop", "index": 0},
		map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "tool_use"}, "usage": map[string]any{"output_tokens": 12}},
		map[string]any{"type": "message_stop"},
	)}
}

func TestGo2LeanRepro_ParameterlessToolChatStreamArgsAreJSON(t *testing.T) {
	conv := NewAnthropicBetaToOpenAIChatConverter(reproH3ParameterlessToolStream(t), "m", false)
	args := map[int]string{}
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		chunk := ev.(wire.ChatStreamChunk)
		for _, ch := range chunk.Choices {
			for _, tc := range ch.Delta.ToolCalls {
				if tc.Function.Arguments != nil {
					args[tc.Index] += *tc.Function.Arguments
				}
			}
		}
	}
	if len(args) == 0 {
		t.Fatal("no tool call streamed")
	}
	for i, a := range args {
		if !json.Valid([]byte(a)) {
			t.Errorf("tool call %d: assembled arguments %q are not JSON (want \"{}\")", i, a)
		}
	}
}

func TestGo2LeanRepro_ParameterlessToolResponsesStreamArgsAreJSON(t *testing.T) {
	conv := NewAnthropicBetaToOpenAIResponsesConverter(reproH3ParameterlessToolStream(t), "m")
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		raw, _ := json.Marshal(ev)
		switch gjson.GetBytes(raw, "type").String() {
		case "response.function_call_arguments.done":
			if a := gjson.GetBytes(raw, "arguments").String(); !json.Valid([]byte(a)) {
				t.Errorf("response.function_call_arguments.done arguments %q are not JSON (want \"{}\")", a)
			}
		case "response.completed":
			if a := gjson.GetBytes(raw, "response.output.0.arguments").String(); !json.Valid([]byte(a)) {
				t.Errorf("response.completed output[0].arguments %q are not JSON (want \"{}\")", a)
			}
		}
	}
}

// findings/responses-anthropic-stream-open-thinking.md: reproH3OneOpenBlock
// drives the Responses -> Anthropic converter and fails when a content
// block starts while another is still open (the Anthropic API streams one
// block at a time) or when a thinking block carries no text.
func reproH3OneOpenBlock(t *testing.T, events []map[string]any) {
	t.Helper()
	stream := openaistream.NewStream[responses.ResponseStreamEventUnion](newFakeResponsesDecoder(eventsToStrings(events)), nil)
	conv := NewOpenAIResponsesToAnthropicConverter(context.Background(), stream, "m")
	open := map[int64]string{}
	thinking := map[int64]string{}
	var trace []string
	var problems []string
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
		raw := ae.RawJSON()
		idx := gjson.Get(raw, "index").Int()
		switch ae.Type {
		case "content_block_start":
			typ := gjson.Get(raw, "content_block.type").String()
			trace = append(trace, fmt.Sprintf("start %d %s", idx, typ))
			if len(open) > 0 {
				problems = append(problems, fmt.Sprintf("block %d (%s) starts while %v is open", idx, typ, open))
			}
			open[idx] = typ
		case "content_block_delta":
			thinking[idx] += gjson.Get(raw, "delta.thinking").String()
		case "content_block_stop":
			trace = append(trace, fmt.Sprintf("stop %d", idx))
			if open[idx] == "thinking" && thinking[idx] == "" {
				problems = append(problems, fmt.Sprintf("thinking block %d is empty", idx))
			}
			delete(open, idx)
		}
	}
	if len(problems) > 0 {
		t.Errorf("%s; blocks: %s", strings.Join(problems, "; "), strings.Join(trace, ", "))
	}
}

func reproH3ResponsesTail() []map[string]any {
	return []map[string]any{
		{"type": "response.output_item.added", "output_index": 1, "item": map[string]any{"id": "msg_1", "type": "message", "role": "assistant", "content": []any{}}},
		{"type": "response.content_part.added", "item_id": "msg_1", "output_index": 1, "content_index": 0, "part": map[string]any{"type": "output_text", "text": ""}},
		{"type": "response.output_text.delta", "item_id": "msg_1", "output_index": 1, "content_index": 0, "delta": "Hello"},
		{"type": "response.output_text.done", "item_id": "msg_1", "output_index": 1, "content_index": 0, "text": "Hello"},
		{"type": "response.content_part.done", "item_id": "msg_1", "output_index": 1, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "Hello"}},
		{"type": "response.output_item.done", "output_index": 1, "item": map[string]any{"id": "msg_1", "type": "message", "role": "assistant"}},
		{"type": "response.completed", "response": map[string]any{"id": "resp_1", "status": "completed",
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}}},
	}
}

// A gpt-5 / o-series stream without reasoning.summary: a reasoning output
// item with no summary text, then the answer.
func TestGo2LeanRepro_ReasoningItemNoSummaryOneOpenBlock(t *testing.T) {
	reproH3OneOpenBlock(t, append([]map[string]any{
		{"type": "response.created", "response": map[string]any{"id": "resp_1"}},
		{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{}}},
		{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{}}},
	}, reproH3ResponsesTail()...))
}

// The same with reasoning.summary "auto" (what the Codex provider requests).
func TestGo2LeanRepro_ReasoningItemWithSummaryOneOpenBlock(t *testing.T) {
	reproH3OneOpenBlock(t, append([]map[string]any{
		{"type": "response.created", "response": map[string]any{"id": "resp_1"}},
		{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{}}},
		{"type": "response.reasoning_summary_part.added", "item_id": "rs_1", "output_index": 0, "summary_index": 0, "part": map[string]any{"type": "summary_text", "text": ""}},
		{"type": "response.reasoning_summary_text.delta", "item_id": "rs_1", "output_index": 0, "summary_index": 0, "delta": "Plan the answer."},
		{"type": "response.reasoning_summary_text.done", "item_id": "rs_1", "output_index": 0, "summary_index": 0, "text": "Plan the answer."},
		{"type": "response.reasoning_summary_part.done", "item_id": "rs_1", "output_index": 0, "summary_index": 0, "part": map[string]any{"type": "summary_text", "text": "Plan the answer."}},
		{"type": "response.output_item.done", "output_index": 0, "item": map[string]any{"id": "rs_1", "type": "reasoning", "summary": []any{map[string]any{"type": "summary_text", "text": "Plan the answer."}}}},
	}, reproH3ResponsesTail()...))
}
