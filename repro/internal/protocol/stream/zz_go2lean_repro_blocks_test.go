package stream

// Reproduction for findings/chat-anthropic-interleaved-blocks.md: the Chat
// -> Anthropic stream converter closes only an open text, thinking or
// refusal block before it starts the next one; a tool_use block stays open
// until the end of the stream. The Anthropic streaming API sends one block
// at a time (content_block_start, its deltas, content_block_stop, then the
// next block), so a client that relies on that order sees parallel tool
// calls, or text after a tool call, start while a tool_use block is open.
// The tests feed realistic Chat chunk JSON (OpenAI's own order for parallel
// tool calls) and fail while the bug is there.

import (
	"fmt"
	"strings"
	"testing"
)

// reproBlocksOneOpen checks the Anthropic event sequence: a block starts
// only when none is open, deltas go to the open block, and every block is
// stopped. It returns the problems found and the block sequence.
func reproBlocksOneOpen(types []string, payloads []map[string]any) (problems []string, seq string) {
	open := -1
	var events []string
	for i, ty := range types {
		switch ty {
		case "content_block_start":
			idx := int(payloads[i]["index"].(float64))
			kind := payloads[i]["content_block"].(map[string]any)["type"].(string)
			events = append(events, fmt.Sprintf("start %d %s", idx, kind))
			if open != -1 {
				problems = append(problems, fmt.Sprintf("block %d (%s) starts while block %d is open", idx, kind, open))
			}
			open = idx
		case "content_block_delta":
			idx := int(payloads[i]["index"].(float64))
			if idx != open {
				problems = append(problems, fmt.Sprintf("delta for block %d while block %d is open", idx, open))
			}
		case "content_block_stop":
			idx := int(payloads[i]["index"].(float64))
			events = append(events, fmt.Sprintf("stop %d", idx))
			if idx == open {
				open = -1
			}
		}
	}
	return problems, strings.Join(events, ", ")
}

func reproBlocksChunk(delta, finish string) string {
	return `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":` +
		delta + `,"logprobs":null,"finish_reason":` + finish + `}]}`
}

// Two parallel tool calls in the order OpenAI streams them: call 0's header
// and arguments, then call 1's header and arguments.
func TestGo2LeanRepro_ParallelToolCallsOneOpenBlock(t *testing.T) {
	types, payloads := reproChatToAnthropic(t,
		reproBlocksChunk(`{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"get_weather","arguments":""}}]}`, "null"),
		reproBlocksChunk(`{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}`, "null"),
		reproBlocksChunk(`{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]}`, "null"),
		reproBlocksChunk(`{"tool_calls":[{"index":1,"id":"call_b","type":"function","function":{"name":"get_weather","arguments":""}}]}`, "null"),
		reproBlocksChunk(`{"tool_calls":[{"index":1,"function":{"arguments":"{\"city\":\"Rome\"}"}}]}`, "null"),
		reproBlocksChunk(`{}`, `"tool_calls"`),
	)
	if problems, seq := reproBlocksOneOpen(types, payloads); len(problems) > 0 {
		t.Errorf("parallel tool calls: %s; blocks: %s (the Anthropic API stops each block before the next starts)", strings.Join(problems, "; "), seq)
	}
}

// Text after a tool call (some OpenAI-compatible providers stream a short
// note after the call): the text block starts while the tool_use block is
// open.
func TestGo2LeanRepro_TextAfterToolCallOneOpenBlock(t *testing.T) {
	types, payloads := reproChatToAnthropic(t,
		reproBlocksChunk(`{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]}`, "null"),
		reproBlocksChunk(`{"content":"Checking the weather."}`, "null"),
		reproBlocksChunk(`{}`, `"tool_calls"`),
	)
	if problems, seq := reproBlocksOneOpen(types, payloads); len(problems) > 0 {
		t.Errorf("text after a tool call: %s; blocks: %s (the Anthropic API stops each block before the next starts)", strings.Join(problems, "; "), seq)
	}
}
