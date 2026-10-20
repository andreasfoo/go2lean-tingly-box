package nonstream

import (
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	"github.com/tingly-dev/tingly-box/internal/protocol/wire"
)

// Hand-written go2lean specs for the non-streaming Anthropic -> OpenAI Chat
// response converter: block counts, text and the stop reason must survive.

// go2lean:spec ConvertAnthropicBetaToOpenAIChat
// SpecConvertAnthropicBetaToOpenAIChatToolCallCount: one tool_call per
// tool_use block, in order, with the same id and name.
func SpecConvertAnthropicBetaToOpenAIChatToolCallCount(bm *anthropic.BetaMessage, responseModel string, ret wire.ChatCompletionWire) bool {
	if bm == nil {
		return true
	}
	if len(ret.Choices) != 1 {
		return false
	}
	calls := ret.Choices[0].Message.ToolCalls
	n := 0
	for _, block := range bm.Content {
		if block.Type != "tool_use" {
			continue
		}
		if n >= len(calls) || calls[n].ID != block.ID || calls[n].Function.Name != block.Name {
			return false
		}
		n++
	}
	return n == len(calls)
}

// go2lean:spec ConvertAnthropicBetaToOpenAIChat
// SpecConvertAnthropicBetaToOpenAIChatTextKept: the message content is the
// concatenation of the text blocks (checked by length).
func SpecConvertAnthropicBetaToOpenAIChatTextKept(bm *anthropic.BetaMessage, responseModel string, ret wire.ChatCompletionWire) bool {
	if bm == nil || len(ret.Choices) != 1 {
		return true
	}
	total := 0
	for _, block := range bm.Content {
		if block.Type == "text" {
			total += len(block.Text)
		}
	}
	return len(ret.Choices[0].Message.Content) == total
}

// go2lean:spec ConvertAnthropicBetaToOpenAIChat
// SpecConvertAnthropicBetaToOpenAIChatFinishReason: max_tokens maps to
// length, tool_use to tool_calls, and the role is carried over.
func SpecConvertAnthropicBetaToOpenAIChatFinishReason(bm *anthropic.BetaMessage, responseModel string, ret wire.ChatCompletionWire) bool {
	if bm == nil || len(ret.Choices) != 1 {
		return true
	}
	fr := ret.Choices[0].FinishReason
	if bm.StopReason == "max_tokens" && fr != "length" {
		return false
	}
	if bm.StopReason == "tool_use" && fr != "tool_calls" {
		return false
	}
	return ret.Choices[0].Message.Role == string(bm.Role) && ret.Model == responseModel && ret.ID == bm.ID
}

// go2lean:spec ConvertOpenAIChatToAnthropicBeta
// SpecConvertOpenAIChatToAnthropicBetaThinkingIsText: the thinking a
// non-streaming Anthropic client reads is the reasoning text the Chat
// answer carries in reasoning_content (the decoded JSON string), and there
// is no thinking when it carries none.
func SpecConvertOpenAIChatToAnthropicBetaThinkingIsText(chat *openai.ChatCompletion, model string, ret0 *anthropic.BetaMessage, ret1 error) bool {
	if chat == nil || ret0 == nil || ret1 != nil || len(chat.Choices) == 0 {
		return true
	}
	want := ""
	if f, ok := chat.Choices[0].Message.JSON.ExtraFields["reasoning_content"]; ok {
		if f.Raw() != "null" && json.Unmarshal([]byte(f.Raw()), &want) != nil {
			return true // not a string: another case
		}
	}
	got := ""
	for _, b := range ret0.Content {
		if b.Type == "thinking" {
			got += b.Thinking
		}
	}
	return got == want
}

// Specs for the hand-found round-2 findings (see benchmark.tsv).

// cutShort says whether an Anthropic stop reason means the output was cut
// short or withheld (not a normal end of turn).
func cutShort(r anthropic.BetaStopReason) bool {
	return r == "max_tokens" || r == "model_context_window_exceeded" || r == "refusal"
}

// go2lean:spec ConvertAnthropicBetaToOpenAIChat
// SpecConvertAnthropicBetaToOpenAIChatCutShortIsNotStop: a stop reason
// meaning the output was cut short or withheld never reaches a Chat client
// as the normal finish_reason "stop".
func SpecConvertAnthropicBetaToOpenAIChatCutShortIsNotStop(bm *anthropic.BetaMessage, responseModel string, ret wire.ChatCompletionWire) bool {
	if bm == nil || len(ret.Choices) != 1 || !cutShort(bm.StopReason) {
		return true
	}
	return ret.Choices[0].FinishReason != "stop"
}

// go2lean:spec ConvertAnthropicBetaToResponsesWire
// SpecConvertAnthropicBetaToResponsesWireCutShortIsIncomplete: a stop
// reason meaning the output was cut short or withheld never reaches a
// Responses client as status "completed".
func SpecConvertAnthropicBetaToResponsesWireCutShortIsIncomplete(resp *anthropic.BetaMessage, responseModel string, actualModel string, ret wire.ResponsesWireResponse) bool {
	if resp == nil || !cutShort(resp.StopReason) {
		return true
	}
	return ret.Status != "completed"
}

// go2lean:spec ConvertOpenAIChatToAnthropicBeta
// SpecConvertOpenAIChatToAnthropicBetaToolUseIDs: every tool_use block an
// Anthropic client reads has a non-empty id (it must answer it by id).
func SpecConvertOpenAIChatToAnthropicBetaToolUseIDs(chat *openai.ChatCompletion, model string, ret0 *anthropic.BetaMessage, ret1 error) bool {
	if ret0 == nil || ret1 != nil {
		return true
	}
	for _, b := range ret0.Content {
		if b.Type == "tool_use" && b.ID == "" {
			return false
		}
	}
	return true
}

// go2lean:spec ConvertChatToResponsesWire
// SpecConvertChatToResponsesWireCallIDs: every function_call item a
// Responses client reads has a non-empty call_id.
func SpecConvertChatToResponsesWireCallIDs(resp *openai.ChatCompletion, responseModel string, actualModel string, ret wire.ResponsesWireResponse) bool {
	for _, item := range ret.Output {
		if item.Type == "function_call" && item.CallID == "" {
			return false
		}
	}
	return true
}

// Specs for the hand-found round-3 findings (see benchmark.tsv).

// go2lean:spec HandleResponsesToAnthropicBeta
// SpecHandleResponsesToAnthropicBetaKeepsRefusal: every non-empty refusal
// in a Responses message reaches the Anthropic client as text.
func SpecHandleResponsesToAnthropicBetaKeepsRefusal(rs *responses.Response, model string, ret anthropic.BetaMessage) bool {
	if rs == nil {
		return true
	}
	text := ""
	for _, b := range ret.Content {
		if b.Type == "text" {
			text += b.Text
		}
	}
	for _, item := range rs.Output {
		for _, c := range item.Content {
			if c.Type == "refusal" && c.Refusal != "" && !strings.Contains(text, c.Refusal) {
				return false
			}
		}
	}
	return true
}

// go2lean:spec HandleResponsesToAnthropicBeta
// SpecHandleResponsesToAnthropicBetaKeepsReasoningSummary: every non-empty
// reasoning summary reaches the Anthropic client as thinking.
func SpecHandleResponsesToAnthropicBetaKeepsReasoningSummary(rs *responses.Response, model string, ret anthropic.BetaMessage) bool {
	if rs == nil {
		return true
	}
	thinking := ""
	for _, b := range ret.Content {
		if b.Type == "thinking" {
			thinking += b.Thinking
		}
	}
	for _, item := range rs.Output {
		if item.Type != "reasoning" {
			continue
		}
		for _, s := range item.Summary {
			if s.Text != "" && !strings.Contains(thinking, s.Text) {
				return false
			}
		}
	}
	return true
}
