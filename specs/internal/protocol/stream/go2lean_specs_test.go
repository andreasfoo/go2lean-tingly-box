package stream

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	"github.com/tingly-dev/tingly-box/internal/protocol"
	"github.com/tingly-dev/tingly-box/internal/protocol/nonstream"
	protocolusage "github.com/tingly-dev/tingly-box/internal/protocol/usage"
	"github.com/tingly-dev/tingly-box/internal/protocol/wire"
)

// Hand-written go2lean specs for the stream helpers behind issue #1924
// (tool-call id truncation) and #1923 (tool-call indexes).

// go2lean:spec truncateToolCallID
// SpecTruncateToolCallIDBounded: the result never exceeds the 40-character
// limit OpenAI imposes on tool_call ids.
func SpecTruncateToolCallIDBounded(id string, ret string) bool {
	return len(ret) <= maxToolCallIDLength
}

// go2lean:spec truncateToolCallID
// SpecTruncateToolCallIDPrefix: the result is a prefix of the input, and the
// input is kept whole when it fits.
func SpecTruncateToolCallIDPrefix(id string, ret string) bool {
	if len(id) <= maxToolCallIDLength {
		return ret == id
	}
	return len(ret) == maxToolCallIDLength && id[:maxToolCallIDLength] == ret
}

// go2lean:spec anthropicStopReasonToIncomplete
// SpecAnthropicStopReasonToIncompleteNonEmpty: max_tokens is an incomplete
// stop with a non-empty reason.
func SpecAnthropicStopReasonToIncompleteNonEmpty(stopReason string, ret0 bool, ret1 string) bool {
	if stopReason == "max_tokens" {
		return ret0 && ret1 != ""
	}
	return true
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventToolCallStartIndex: the chunk
// that opens a tool call carries the index of that tool call among the
// tool calls of the message (issue #1923: a second parallel tool call must
// not reuse index 0).
func SpecAnthropicToOpenAIConverterProcessEventToolCallStartIndex(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if c == nil || event == nil {
		return true
	}
	if event.Type != "content_block_start" || event.ContentBlock.Type != "tool_use" {
		return true
	}
	n := len(c.pending)
	if n == 0 {
		return true
	}
	last, isChunk := c.pending[n-1].(wire.ChatStreamChunk)
	if !isChunk {
		return true
	}
	if len(last.Choices) == 0 || len(last.Choices[0].Delta.ToolCalls) == 0 {
		return true
	}
	tc := last.Choices[0].Delta.ToolCalls[0]
	if tc.ID != event.ContentBlock.ID {
		return true
	}
	return tc.Index == len(c.pendingToolCalls)
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventToolCallDeltaIndex: an argument
// delta chunk carries the index of the tool call it belongs to.
func SpecAnthropicToOpenAIConverterProcessEventToolCallDeltaIndex(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if c == nil || event == nil {
		return true
	}
	if event.Type != "content_block_delta" || event.Delta.Type != "input_json_delta" || event.Delta.PartialJSON == "" {
		return true
	}
	if c.toolCallID == "" {
		return true
	}
	n := len(c.pending)
	if n == 0 {
		return true
	}
	last, isChunk := c.pending[n-1].(wire.ChatStreamChunk)
	if !isChunk {
		return true
	}
	if len(last.Choices) == 0 || len(last.Choices[0].Delta.ToolCalls) == 0 {
		return true
	}
	tc := last.Choices[0].Delta.ToolCalls[0]
	if tc.Function.Arguments == nil || *tc.Function.Arguments != event.Delta.PartialJSON {
		return true
	}
	return tc.Index == len(c.pendingToolCalls)
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventMaxTokensIsLength: whatever
// state the converter is in, a message_delta whose stop reason is
// max_tokens followed by message_stop ends the stream with finish_reason
// "length", as the non-streaming converter does (issue #1923, second part).
func SpecAnthropicToOpenAIConverterProcessEventMaxTokensIsLength(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if c == nil || c.done {
		return true
	}
	delta := anthropic.BetaRawMessageStreamEventUnion{Type: "message_delta"}
	delta.Delta.StopReason = "max_tokens"
	c.processEvent(&delta)
	stop := anthropic.BetaRawMessageStreamEventUnion{Type: "message_stop"}
	n0 := len(c.pending)
	c.processEvent(&stop)
	if c.hookErr != nil || len(c.pending) == n0 {
		return true
	}
	last, isChunk := c.pending[len(c.pending)-1].(wire.ChatStreamChunk)
	if !isChunk || len(last.Choices) == 0 || last.Choices[0].FinishReason == nil {
		return false
	}
	return *last.Choices[0].FinishReason == "length"
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventParallelToolCallIndexes: two
// tool calls started one after the other in the same message get distinct
// tool_calls[].index values, whatever state the converter was in (issue
// #1923: OpenAI clients merge tool calls that share an index).
func SpecAnthropicToOpenAIConverterProcessEventParallelToolCallIndexes(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if c == nil {
		return true
	}
	stop := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_stop"}
	c.processEvent(&stop) // close whatever block the random event opened
	n0 := len(c.pending)
	first := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_start"}
	first.ContentBlock.Type = "tool_use"
	first.ContentBlock.ID = "toolu_first"
	first.ContentBlock.Name = "bash"
	c.processEvent(&first)
	c.processEvent(&stop)
	second := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_start"}
	second.ContentBlock.Type = "tool_use"
	second.ContentBlock.ID = "toolu_second"
	second.ContentBlock.Name = "bash"
	c.processEvent(&second)
	firstIdx, secondIdx := -1, -1
	for _, p := range c.pending[n0:] {
		chunk, isChunk := p.(wire.ChatStreamChunk)
		if !isChunk || len(chunk.Choices) == 0 {
			continue
		}
		for _, tc := range chunk.Choices[0].Delta.ToolCalls {
			if tc.ID == "toolu_first" {
				firstIdx = tc.Index
			}
			if tc.ID == "toolu_second" {
				secondIdx = tc.Index
			}
		}
	}
	if firstIdx < 0 || secondIdx < 0 {
		return true // a hook suppressed one of them
	}
	return firstIdx != secondIdx
}

// Streaming and non-streaming consistency: an OpenAI Chat client that asks
// for a stream should receive what a non-streaming client receives for
// the same upstream message. The specs replay the message carried by the
// random event as an Anthropic event stream through a fresh converter and
// compare with nonstream.ConvertAnthropicBetaToOpenAIChat on the same
// message. Blocks other than text, thinking and tool_use are skipped on
// both sides.

// replayed is what the stream converter emitted for one message.
type replayed struct {
	text, reasoning, finish string
	calls                   []string
}

// replayMessage feeds bm to a fresh converter as message_start, one
// start/delta/stop triple per block, message_delta with bm's stop reason
// and message_stop, and collects the chunks.
func replayMessage(bm anthropic.BetaMessage) replayed {
	c := newAnthropicToOpenAIConverter(nil, "m", true, nil)
	start := anthropic.BetaRawMessageStreamEventUnion{Type: "message_start"}
	start.Message = bm
	c.processEvent(&start)
	for i, b := range bm.Content {
		if b.Type != "text" && b.Type != "thinking" && b.Type != "tool_use" {
			continue
		}
		cbs := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_start"}
		cbs.Index = int64(i)
		cbs.ContentBlock.Type = b.Type
		cbs.ContentBlock.ID = b.ID
		cbs.ContentBlock.Name = b.Name
		c.processEvent(&cbs)
		delta := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_delta"}
		delta.Index = int64(i)
		if b.Type == "text" {
			delta.Delta.Type = "text_delta"
			delta.Delta.Text = b.Text
			c.processEvent(&delta)
		}
		if b.Type == "thinking" {
			delta.Delta.Type = "thinking_delta"
			delta.Delta.Thinking = b.Thinking
			c.processEvent(&delta)
		}
		cbstop := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_stop"}
		cbstop.Index = int64(i)
		c.processEvent(&cbstop)
	}
	md := anthropic.BetaRawMessageStreamEventUnion{Type: "message_delta"}
	md.Delta.StopReason = bm.StopReason
	c.processEvent(&md)
	stop := anthropic.BetaRawMessageStreamEventUnion{Type: "message_stop"}
	c.processEvent(&stop)
	var r replayed
	for _, p := range c.pending {
		chunk, isChunk := p.(wire.ChatStreamChunk)
		if !isChunk || len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		r.text += ch.Delta.Content
		r.reasoning += ch.Delta.ReasoningContent
		for _, tc := range ch.Delta.ToolCalls {
			if tc.Type == "function" { // the chunk that opens a tool call
				r.calls = append(r.calls, tc.ID+"/"+tc.Function.Name)
			}
		}
		if ch.FinishReason != nil {
			r.finish = *ch.FinishReason
		}
	}
	return r
}

// nonStreamed is what the non-streaming converter returns for bm.
func nonStreamed(bm anthropic.BetaMessage) (text, reasoning, finish string, calls []string, ok bool) {
	ns := nonstream.ConvertAnthropicBetaToOpenAIChat(&bm, "m")
	if len(ns.Choices) != 1 {
		return "", "", "", nil, false
	}
	msg := ns.Choices[0].Message
	for _, tc := range msg.ToolCalls {
		calls = append(calls, tc.ID+"/"+tc.Function.Name)
	}
	return msg.Content, msg.ReasoningContent, ns.Choices[0].FinishReason, calls, true
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventStreamTextMatches: the
// streamed content equals the non-streamed message content.
func SpecAnthropicToOpenAIConverterProcessEventStreamTextMatches(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil {
		return true
	}
	text, _, _, _, ok := nonStreamed(event.Message)
	if !ok {
		return true
	}
	return replayMessage(event.Message).text == text
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventStreamReasoningMatches: the
// streamed reasoning_content equals the non-streamed one, for messages
// whose thinking blocks have the shape the API sends (thinking text in
// Thinking, Text empty).
func SpecAnthropicToOpenAIConverterProcessEventStreamReasoningMatches(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil {
		return true
	}
	for _, b := range event.Message.Content {
		if b.Type == "thinking" && b.Text != "" {
			return true
		}
	}
	_, reasoning, _, _, ok := nonStreamed(event.Message)
	if !ok {
		return true
	}
	return replayMessage(event.Message).reasoning == reasoning
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventStreamToolCallsMatch: the
// streamed tool calls (id and name, in order) equal the non-streamed ones.
func SpecAnthropicToOpenAIConverterProcessEventStreamToolCallsMatch(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil {
		return true
	}
	_, _, _, calls, ok := nonStreamed(event.Message)
	if !ok {
		return true
	}
	got := replayMessage(event.Message).calls
	if len(got) != len(calls) {
		return false
	}
	for i := range got {
		if got[i] != calls[i] {
			return false
		}
	}
	return true
}

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventStreamFinishMatches: the
// streamed finish_reason equals the non-streamed one (max_tokens excluded:
// already confirmed by MaxTokensIsLength), for messages whose stop reason
// agrees with their blocks.
func SpecAnthropicToOpenAIConverterProcessEventStreamFinishMatches(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil || event.Message.StopReason == "max_tokens" {
		return true
	}
	// As the API sends it: stop reason tool_use exactly when the message
	// holds a tool_use block.
	hasToolUse := false
	for _, b := range event.Message.Content {
		if b.Type == "tool_use" {
			hasToolUse = true
		}
	}
	if hasToolUse != (event.Message.StopReason == "tool_use") {
		return true
	}
	_, _, finish, _, ok := nonStreamed(event.Message)
	if !ok {
		return true
	}
	return replayMessage(event.Message).finish == finish
}

// Anthropic -> Responses: the response.completed (or .incomplete) event
// of the stream carries a whole response, compared with
// nonstream.ConvertAnthropicBetaToResponsesWire on the same message.

// apiShaped: thinking text in Thinking (Text empty), stop reason tool_use
// exactly when a tool_use block is present.
func apiShaped(bm anthropic.BetaMessage) bool {
	hasToolUse := false
	for _, b := range bm.Content {
		if b.Type == "thinking" && b.Text != "" {
			return false
		}
		if b.Type == "tool_use" {
			hasToolUse = true
		}
	}
	return hasToolUse == (bm.StopReason == "tool_use")
}

// replayToResponses feeds bm to a fresh Anthropic -> Responses converter
// as an event stream and returns the response of its final event.
func replayToResponses(bm anthropic.BetaMessage) (wire.ResponsesWireResponse, bool) {
	c := newAnthropicBetaToResponsesConverter(nil, "m")
	start := anthropic.BetaRawMessageStreamEventUnion{Type: "message_start"}
	start.Message = bm
	c.processEvent(&start)
	for i, b := range bm.Content {
		if b.Type != "text" && b.Type != "thinking" && b.Type != "tool_use" {
			continue
		}
		cbs := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_start"}
		cbs.Index = int64(i)
		cbs.ContentBlock.Type = b.Type
		cbs.ContentBlock.ID = b.ID
		cbs.ContentBlock.Name = b.Name
		c.processEvent(&cbs)
		delta := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_delta"}
		delta.Index = int64(i)
		if b.Type == "text" {
			delta.Delta.Type = "text_delta"
			delta.Delta.Text = b.Text
			c.processEvent(&delta)
		}
		if b.Type == "thinking" {
			delta.Delta.Type = "thinking_delta"
			delta.Delta.Thinking = b.Thinking
			c.processEvent(&delta)
		}
		cbstop := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_stop"}
		cbstop.Index = int64(i)
		c.processEvent(&cbstop)
	}
	md := anthropic.BetaRawMessageStreamEventUnion{Type: "message_delta"}
	md.Delta.StopReason = bm.StopReason
	c.processEvent(&md)
	stop := anthropic.BetaRawMessageStreamEventUnion{Type: "message_stop"}
	c.processEvent(&stop)
	var last wire.ResponsesWireResponse
	found := false
	for _, p := range c.pending {
		if e, ok := p.(wire.ResponsesCompletedEvent); ok {
			last, found = e.Response, true
		}
		if e, ok := p.(wire.ResponsesIncompleteEvent); ok {
			last, found = e.Response, true
		}
	}
	return last, found
}

// responseSummary is what a Responses client reads from a response: the
// status, the incomplete reason, the message text and the function calls.
func responseSummary(r wire.ResponsesWireResponse) (status, reason, text string, calls []string) {
	status = r.Status
	if r.IncompleteDetails != nil {
		reason = r.IncompleteDetails.Reason
	}
	for _, item := range r.Output {
		if item.Type == "message" {
			for _, part := range item.Content {
				if part.Type == "output_text" {
					text += part.Text
				}
			}
		}
		if item.Type == "function_call" {
			calls = append(calls, item.CallID+"/"+item.Name)
		}
	}
	return
}

// go2lean:spec anthropicBetaToResponsesConverter.processEvent
// SpecAnthropicBetaToResponsesConverterProcessEventStreamTextMatches: the
// streamed final response has the non-streamed message text.
func SpecAnthropicBetaToResponsesConverterProcessEventStreamTextMatches(c *anthropicBetaToResponsesConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil || !apiShaped(event.Message) {
		return true
	}
	streamed, ok := replayToResponses(event.Message)
	if !ok {
		return false
	}
	_, _, t1, _ := responseSummary(streamed)
	_, _, t2, _ := responseSummary(nonstream.ConvertAnthropicBetaToResponsesWire(&event.Message, "m", "m"))
	go2leanProbe("streamed", t1)
	go2leanProbe("nonstreamed", t2)
	return t1 == t2
}

// go2lean:spec anthropicBetaToResponsesConverter.processEvent
// SpecAnthropicBetaToResponsesConverterProcessEventStreamCallsMatch: the
// streamed final response has the non-streamed function calls.
func SpecAnthropicBetaToResponsesConverterProcessEventStreamCallsMatch(c *anthropicBetaToResponsesConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil || !apiShaped(event.Message) {
		return true
	}
	streamed, ok := replayToResponses(event.Message)
	if !ok {
		return false
	}
	_, _, _, c1 := responseSummary(streamed)
	_, _, _, c2 := responseSummary(nonstream.ConvertAnthropicBetaToResponsesWire(&event.Message, "m", "m"))
	if len(c1) != len(c2) {
		return false
	}
	for i := range c1 {
		if c1[i] != c2[i] {
			return false
		}
	}
	return true
}

// go2lean:spec anthropicBetaToResponsesConverter.processEvent
// SpecAnthropicBetaToResponsesConverterProcessEventStreamStatusMatches: the
// streamed final response has the non-streamed status and incomplete
// reason (max_tokens included).
func SpecAnthropicBetaToResponsesConverterProcessEventStreamStatusMatches(c *anthropicBetaToResponsesConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil || !apiShaped(event.Message) {
		return true
	}
	streamed, ok := replayToResponses(event.Message)
	if !ok {
		return false
	}
	s1, r1, _, _ := responseSummary(streamed)
	s2, r2, _, _ := responseSummary(nonstream.ConvertAnthropicBetaToResponsesWire(&event.Message, "m", "m"))
	return s1 == s2 && r1 == r2
}

// ----- Responses -> Chat: streamed vs non-streamed -----

// chatShaped keeps Responses answers a Responses API produces and the Chat
// converters claim to handle: messages of output_text and refusal parts,
// and function calls with a call id.
func chatShaped(r responses.Response) bool {
	for _, item := range r.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type != "output_text" && part.Type != "refusal" {
					return false
				}
			}
		case "function_call":
			if item.CallID == "" || item.ID == "" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// replayResponsesToChat feeds r to a fresh Responses -> Chat converter as
// the event stream the Responses API sends for it and returns what a Chat
// client assembles from the chunks: content, the tool calls as "id/name"
// and the finish reason. (A Chat stream chunk has no refusal field.)
func replayResponsesToChat(r responses.Response) (text, finish string, calls []string) {
	c := newResponsesToChatConverter(nil, "m", true)
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.created", Response: r})
	for i, item := range r.Output {
		c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.added", Item: item, OutputIndex: int64(i)})
		if item.Type == "message" {
			for _, part := range item.Content {
				if part.Type == "output_text" {
					c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_text.delta", ItemID: item.ID, OutputIndex: int64(i), Delta: part.Text})
				}
				if part.Type == "refusal" {
					c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.refusal.delta", ItemID: item.ID, OutputIndex: int64(i), Delta: part.Refusal})
				}
			}
		}
		if item.Type == "function_call" {
			c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.function_call_arguments.delta", ItemID: item.ID, OutputIndex: int64(i), Delta: item.Arguments.OfString})
		}
		c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.done", Item: item, OutputIndex: int64(i)})
	}
	terminal := "response.completed"
	if r.Status == "incomplete" {
		terminal = "response.incomplete"
	}
	c.processEvent(&responses.ResponseStreamEventUnion{Type: terminal, Response: r})
	for _, p := range c.pending {
		chunk, ok := p.(wire.ChatStreamChunk)
		if !ok {
			continue
		}
		for _, ch := range chunk.Choices {
			text += ch.Delta.Content
			for _, tc := range ch.Delta.ToolCalls {
				if tc.ID != "" {
					calls = append(calls, tc.ID+"/"+tc.Function.Name)
				}
			}
			if ch.FinishReason != nil {
				finish = *ch.FinishReason
			}
		}
	}
	return
}

// nonStreamedChat is the same answer converted without streaming.
func nonStreamedChat(r responses.Response) (text, refusal, finish string, calls []string) {
	w := nonstream.ConvertResponsesToOpenAIChat(&r, "m")
	if len(w.Choices) == 0 {
		return
	}
	m := w.Choices[0].Message
	for _, tc := range m.ToolCalls {
		calls = append(calls, tc.ID+"/"+tc.Function.Name)
	}
	return m.Content, m.Refusal, w.Choices[0].FinishReason, calls
}

// go2lean:spec responsesToChatConverter.processEvent
// SpecResponsesToChatConverterProcessEventStreamTextMatches: a Chat client
// streaming a Responses answer reads the content a non-streaming one does.
func SpecResponsesToChatConverterProcessEventStreamTextMatches(c *responsesToChatConverter, evt *responses.ResponseStreamEventUnion) bool {
	if evt == nil || !chatShaped(evt.Response) {
		return true
	}
	t1, _, _ := replayResponsesToChat(evt.Response)
	t2, _, _, _ := nonStreamedChat(evt.Response)
	go2leanProbe("streamed", t1)
	go2leanProbe("nonstreamed", t2)
	return t1 == t2
}

// go2lean:spec responsesToChatConverter.processEvent
// SpecResponsesToChatConverterProcessEventStreamRefusalDelivered: a refusal
// a non-streaming Chat client reads (message.refusal) reaches a streaming
// one too, in whatever field.
func SpecResponsesToChatConverterProcessEventStreamRefusalDelivered(c *responsesToChatConverter, evt *responses.ResponseStreamEventUnion) bool {
	if evt == nil || !chatShaped(evt.Response) {
		return true
	}
	t1, _, _ := replayResponsesToChat(evt.Response)
	_, r2, _, _ := nonStreamedChat(evt.Response)
	go2leanProbe("streamed", t1)
	go2leanProbe("nonstreamed-refusal", r2)
	return r2 == "" || strings.Contains(t1, r2)
}

// go2lean:spec responsesToChatConverter.processEvent
// SpecResponsesToChatConverterProcessEventStreamCallsMatch: the streamed
// tool calls (id/name, in order) are the non-streamed ones.
func SpecResponsesToChatConverterProcessEventStreamCallsMatch(c *responsesToChatConverter, evt *responses.ResponseStreamEventUnion) bool {
	if evt == nil || !chatShaped(evt.Response) {
		return true
	}
	_, _, c1 := replayResponsesToChat(evt.Response)
	_, _, _, c2 := nonStreamedChat(evt.Response)
	if len(c1) != len(c2) {
		return false
	}
	for i := range c1 {
		if c1[i] != c2[i] {
			return false
		}
	}
	return true
}

// replayResponsesToChatScheduled is replayResponsesToChat over another
// legal event schedule of the same response: the arguments of function
// call i go out as one delta when bit i of schedule is set and only in
// the .done events otherwise (the API allows both), and every call gets
// its response.function_call_arguments.done. It returns the arguments a
// Chat client assembles per tool call, in call order.
func replayResponsesToChatScheduled(r responses.Response, schedule int64) (args []string) {
	c := newResponsesToChatConverter(nil, "m", true)
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.created", Response: r})
	for i, item := range r.Output {
		c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.added", Item: item, OutputIndex: int64(i)})
		if item.Type == "message" {
			for _, part := range item.Content {
				if part.Type == "output_text" {
					c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_text.delta", ItemID: item.ID, OutputIndex: int64(i), Delta: part.Text})
				}
			}
		}
		if item.Type == "function_call" {
			if schedule&(1<<uint(i%62)) != 0 {
				c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.function_call_arguments.delta", ItemID: item.ID, OutputIndex: int64(i), Delta: item.Arguments.OfString})
			}
			c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.function_call_arguments.done", ItemID: item.ID, OutputIndex: int64(i), Arguments: item.Arguments.OfString})
		}
		c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.done", Item: item, OutputIndex: int64(i)})
	}
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.completed", Response: r})
	byIndex := map[int]string{}
	var order []int
	for _, p := range c.pending {
		chunk, ok := p.(wire.ChatStreamChunk)
		if !ok {
			continue
		}
		for _, ch := range chunk.Choices {
			for _, tc := range ch.Delta.ToolCalls {
				if _, seen := byIndex[tc.Index]; !seen {
					order = append(order, tc.Index)
					byIndex[tc.Index] = ""
				}
				if tc.Function.Arguments != nil {
					byIndex[tc.Index] += *tc.Function.Arguments
				}
			}
		}
	}
	for _, i := range order {
		args = append(args, byIndex[i])
	}
	return
}

// go2lean:spec responsesToChatConverter.processEvent
// SpecResponsesToChatConverterProcessEventArgumentsAnySchedule: whichever
// legal schedule the arguments come in (the event's sequence number picks
// one), a Chat client assembles each call's arguments as the response
// has them.
func SpecResponsesToChatConverterProcessEventArgumentsAnySchedule(c *responsesToChatConverter, evt *responses.ResponseStreamEventUnion) bool {
	if evt == nil || !chatShaped(evt.Response) || evt.Response.Status == "incomplete" {
		return true
	}
	var want []string
	for _, item := range evt.Response.Output {
		if item.Type == "function_call" {
			want = append(want, item.Arguments.OfString)
		}
	}
	got := replayResponsesToChatScheduled(evt.Response, evt.SequenceNumber)
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		go2leanProbe("streamed-args", got[i])
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// go2lean:spec responsesToChatConverter.processEvent
// SpecResponsesToChatConverterProcessEventStreamFinishMatches: the streamed
// finish reason is the non-streamed one.
func SpecResponsesToChatConverterProcessEventStreamFinishMatches(c *responsesToChatConverter, evt *responses.ResponseStreamEventUnion) bool {
	if evt == nil || !chatShaped(evt.Response) {
		return true
	}
	_, f1, _ := replayResponsesToChat(evt.Response)
	_, _, f2, _ := nonStreamedChat(evt.Response)
	go2leanProbe("streamed", f1)
	go2leanProbe("nonstreamed", f2)
	return f1 == f2
}

// oneOpenBlock feeds the chunk's choices to a fresh Chat -> Anthropic
// converter one per chunk (finish reasons cleared: the stream's end is not
// under test) and checks that one content block is open at a time and
// every delta goes to it. With noTools, a sequence holding a tool call
// passes untested.
func oneOpenBlock(chunk *openai.ChatCompletionChunk, noTools bool) bool {
	if chunk == nil {
		return true
	}
	// The converter newOpenAIToAnthropicConverter builds when its token
	// counter cannot be made (it then counts from upstream usage only): the
	// tiktoken tables are out of the model's reach.
	conv := &openAIToAnthropicConverter{responseModel: "m", mapFinishReason: mapOpenAIFinishReasonToAnthropic,
		messageID: "msg_go2lean", state: newStreamState(), usage: protocol.ZeroTokenUsage()}
	for _, ch := range chunk.Choices {
		ch.FinishReason = "" // the stream's end is not under test
		if noTools && len(ch.Delta.ToolCalls) > 0 {
			return true
		}
		conv.processChunk(&openai.ChatCompletionChunk{Choices: []openai.ChatCompletionChunkChoice{ch}})
	}
	open := -1
	for _, p := range conv.pending {
		ev, ok := p.(anthropicStreamEvent)
		if !ok {
			continue
		}
		switch d := ev.data.(type) {
		case anthropicContentBlockStartEvent:
			if open != -1 {
				return false
			}
			open = d.Index
		case anthropicContentBlockDeltaEvent:
			if d.Index != open {
				return false
			}
		case anthropicContentBlockStopEvent:
			if d.Index != open {
				return false
			}
			open = -1
		}
	}
	return true
}

// go2lean:spec openAIToAnthropicConverter.processChunk
// SpecOpenAIToAnthropicConverterProcessChunkOneOpenBlock: whatever deltas a
// Chat stream sends, the Anthropic events keep one content block open at a
// time and send every delta to the open block, as the Anthropic API does.
func SpecOpenAIToAnthropicConverterProcessChunkOneOpenBlock(c *openAIToAnthropicConverter, chunk *openai.ChatCompletionChunk) bool {
	return oneOpenBlock(chunk, false)
}

// go2lean:spec openAIToAnthropicConverter.processChunk
// SpecOpenAIToAnthropicConverterProcessChunkOneOpenTextBlock: the same for
// streams of text, refusals and reasoning only (no tool calls).
func SpecOpenAIToAnthropicConverterProcessChunkOneOpenTextBlock(c *openAIToAnthropicConverter, chunk *openai.ChatCompletionChunk) bool {
	return oneOpenBlock(chunk, true)
}

// go2leanProbe records v under label: go2lean's model keeps it in the
// counterexample, the real run prints it, so the two can be compared.
func go2leanProbe(label string, v string) {
	fmt.Printf("GO2LEAN_PROBE %s %q\n", label, v)
}

// ----- Specs for the hand-found round-2 findings (see benchmark.tsv) -----

// go2lean:spec chatToResponsesConverter.processChunk
// SpecChatToResponsesConverterProcessChunkTerminalUsage: a Chat stream of
// the chunk's choices (one per chunk, without tool calls) with its usage in
// one more chunk, at any position (the chunk's created time picks it): the
// usage on the terminal Responses event is that usage.
func SpecChatToResponsesConverterProcessChunkTerminalUsage(c *chatToResponsesConverter, chunk *openai.ChatCompletionChunk) bool {
	if chunk == nil || !chunkHasUsage(chunk.Usage) || len(chunk.Choices) == 0 {
		return true
	}
	for _, ch := range chunk.Choices {
		if len(ch.Delta.ToolCalls) > 0 {
			return true // tool calls mint random item ids
		}
	}
	conv := &chatToResponsesConverter{responseModel: "m", responseID: "resp_go2lean", textItemID: "msg_go2lean",
		textOutputIndex: -1, usage: protocol.ZeroTokenUsage(), pendingToolCalls: map[int]*pendingToolCallResponse{}}
	n := int64(len(chunk.Choices))
	at := chunk.Created % (n + 1)
	if at < 0 {
		at = -at
	}
	usageChunk := &openai.ChatCompletionChunk{Usage: chunk.Usage}
	for i, ch := range chunk.Choices {
		if int64(i) == at {
			conv.processChunk(usageChunk)
		}
		conv.processChunk(&openai.ChatCompletionChunk{Choices: []openai.ChatCompletionChunkChoice{ch}})
	}
	if at == n {
		conv.processChunk(usageChunk)
	}
	if !conv.completedSent {
		return true // no finish reason: the stream fails, no terminal event
	}
	var got *wire.ResponsesUsageWire
	for _, p := range conv.pending {
		switch e := p.(type) {
		case wire.ResponsesCompletedEvent:
			got = e.Response.Usage
		case wire.ResponsesIncompleteEvent:
			got = e.Response.Usage
		}
	}
	want := protocolusage.ToResponsesUsageWire(protocolusage.FromOpenAIChatCompletion(chunk.Usage))
	return got != nil && got.InputTokens == want.InputTokens && got.OutputTokens == want.OutputTokens
}

// streamedToolIndexes replays r to a Responses -> Chat converter as
// replayResponsesToChat does and returns the tool_calls[].index values in
// order of first appearance.
func streamedToolIndexes(r responses.Response) (indexes []int) {
	c := newResponsesToChatConverter(nil, "m", true)
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.created", Response: r})
	for i, item := range r.Output {
		c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.added", Item: item, OutputIndex: int64(i)})
		if item.Type == "function_call" {
			c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.function_call_arguments.delta", ItemID: item.ID, OutputIndex: int64(i), Delta: item.Arguments.OfString})
		}
		c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.done", Item: item, OutputIndex: int64(i)})
	}
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.completed", Response: r})
	seen := map[int]bool{}
	for _, p := range c.pending {
		chunk, ok := p.(wire.ChatStreamChunk)
		if !ok {
			continue
		}
		for _, ch := range chunk.Choices {
			for _, tc := range ch.Delta.ToolCalls {
				if !seen[tc.Index] {
					seen[tc.Index] = true
					indexes = append(indexes, tc.Index)
				}
			}
		}
	}
	return
}

// go2lean:spec responsesToChatConverter.processEvent
// SpecResponsesToChatConverterProcessEventToolIndexesDense: a Chat client
// streaming a Responses answer with n function calls (next to messages and
// reasoning items) sees the tool_calls indexes 0..n-1 in order, as Chat
// clients accumulate calls by index.
func SpecResponsesToChatConverterProcessEventToolIndexesDense(c *responsesToChatConverter, evt *responses.ResponseStreamEventUnion) bool {
	if evt == nil {
		return true
	}
	calls := 0
	for _, item := range evt.Response.Output {
		switch item.Type {
		case "message", "reasoning":
		case "function_call":
			if item.CallID == "" || item.ID == "" {
				return true
			}
			calls++
		default:
			return true
		}
	}
	got := streamedToolIndexes(evt.Response)
	if len(got) != calls {
		return false
	}
	for i, idx := range got {
		if idx != i {
			return false
		}
	}
	return true
}

// go2lean:spec openAIToAnthropicConverter.processChunk
// SpecOpenAIToAnthropicConverterProcessChunkToolUseIDs: whatever tool-call
// deltas a Chat stream sends, every tool_use block an Anthropic client
// reads has a non-empty id.
func SpecOpenAIToAnthropicConverterProcessChunkToolUseIDs(c *openAIToAnthropicConverter, chunk *openai.ChatCompletionChunk) bool {
	if chunk == nil {
		return true
	}
	conv := &openAIToAnthropicConverter{responseModel: "m", mapFinishReason: mapOpenAIFinishReasonToAnthropic,
		messageID: "msg_go2lean", state: newStreamState(), usage: protocol.ZeroTokenUsage()}
	for _, ch := range chunk.Choices {
		ch.FinishReason = ""
		conv.processChunk(&openai.ChatCompletionChunk{Choices: []openai.ChatCompletionChunkChoice{ch}})
	}
	for _, p := range conv.pending {
		ev, ok := p.(anthropicStreamEvent)
		if !ok {
			continue
		}
		if d, ok := ev.data.(anthropicContentBlockStartEvent); ok && d.ContentBlock.Type == "tool_use" {
			if d.ContentBlock.ID == nil || *d.ContentBlock.ID == "" {
				return false
			}
		}
	}
	return true
}

// ----- Specs for the hand-found round-3 findings (see benchmark.tsv) -----

// go2lean:spec anthropicToOpenAIConverter.processEvent
// SpecAnthropicToOpenAIConverterProcessEventToolArgsAreJSON: replaying the
// message's tool calls as the Anthropic API streams them (the input as one
// input_json_delta; an empty input as an empty delta), every tool call a
// Chat client assembles has arguments that decode as a JSON object.
func SpecAnthropicToOpenAIConverterProcessEventToolArgsAreJSON(c *anthropicToOpenAIConverter, event *anthropic.BetaRawMessageStreamEventUnion) bool {
	if event == nil {
		return true
	}
	bm := event.Message
	conv := newAnthropicToOpenAIConverter(nil, "m", true, nil)
	start := anthropic.BetaRawMessageStreamEventUnion{Type: "message_start"}
	start.Message = bm
	conv.processEvent(&start)
	calls := 0
	for i, b := range bm.Content {
		if b.Type != "tool_use" {
			continue
		}
		var input map[string]any
		if json.Unmarshal(b.Input, &input) != nil {
			return true // not an object: the API never sends it
		}
		partial := string(b.Input)
		if len(input) == 0 {
			partial = ""
		}
		calls++
		cbs := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_start"}
		cbs.Index = int64(i)
		cbs.ContentBlock.Type = "tool_use"
		cbs.ContentBlock.ID = b.ID
		cbs.ContentBlock.Name = b.Name
		conv.processEvent(&cbs)
		delta := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_delta"}
		delta.Index = int64(i)
		delta.Delta.Type = "input_json_delta"
		delta.Delta.PartialJSON = partial
		conv.processEvent(&delta)
		cbstop := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_stop"}
		cbstop.Index = int64(i)
		conv.processEvent(&cbstop)
	}
	var args []string
	for _, p := range conv.pending {
		chunk, ok := p.(wire.ChatStreamChunk)
		if !ok || len(chunk.Choices) == 0 {
			continue
		}
		for _, tc := range chunk.Choices[0].Delta.ToolCalls {
			if tc.Type == "function" {
				args = append(args, "")
			}
			if tc.Function.Arguments != nil && len(args) > 0 {
				args[len(args)-1] += *tc.Function.Arguments
			}
		}
	}
	if len(args) != calls {
		return true // a hook suppressed a call
	}
	for _, a := range args {
		go2leanProbe("args", a)
		var obj map[string]any
		if json.Unmarshal([]byte(a), &obj) != nil {
			return false
		}
	}
	return true
}
