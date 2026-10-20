package stream

// Reproductions of go2lean findings on the stream converters. Each test
// asserts the behaviour a client expects: it fails while the bug is there.

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3/responses"

	"github.com/tingly-dev/tingly-box/internal/protocol/nonstream"
	"github.com/tingly-dev/tingly-box/internal/protocol/wire"
)

func reproChunks(pending []interface{}) []wire.ChatStreamChunk {
	var out []wire.ChatStreamChunk
	for _, p := range pending {
		if c, ok := p.(wire.ChatStreamChunk); ok {
			out = append(out, c)
		}
	}
	return out
}

// findings/issue-1923-stream-indexes.md (parallel tool calls)
func TestGo2LeanRepro_ParallelToolCallsGetDistinctIndexes(t *testing.T) {
	c := newAnthropicToOpenAIConverter(nil, "m", true, nil)
	for _, id := range []string{"toolu_first", "toolu_second"} {
		start := anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_start"}
		start.ContentBlock.Type = "tool_use"
		start.ContentBlock.ID = id
		start.ContentBlock.Name = "bash"
		c.processEvent(&start)
		c.processEvent(&anthropic.BetaRawMessageStreamEventUnion{Type: "content_block_stop"})
	}
	index := map[string]int{}
	for _, ch := range reproChunks(c.pending) {
		for _, choice := range ch.Choices {
			for _, tc := range choice.Delta.ToolCalls {
				if tc.ID != "" {
					index[tc.ID] = tc.Index
				}
			}
		}
	}
	if index["toolu_first"] == index["toolu_second"] {
		t.Errorf("both tool calls stream with tool_calls[].index %d; a Chat client merges them", index["toolu_first"])
	}
}

// findings/issue-1923-stream-indexes.md (max_tokens)
func TestGo2LeanRepro_MaxTokensFinishesWithLength(t *testing.T) {
	c := newAnthropicToOpenAIConverter(nil, "m", true, nil)
	delta := anthropic.BetaRawMessageStreamEventUnion{Type: "message_delta"}
	delta.Delta.StopReason = "max_tokens"
	c.processEvent(&delta)
	c.processEvent(&anthropic.BetaRawMessageStreamEventUnion{Type: "message_stop"})
	finish := ""
	for _, ch := range reproChunks(c.pending) {
		for _, choice := range ch.Choices {
			if choice.FinishReason != nil {
				finish = *choice.FinishReason
			}
		}
	}
	if finish != "length" {
		t.Errorf("stop reason max_tokens streams finish_reason %q, want \"length\"", finish)
	}
}

// findings/responses-chat-stream-refusal.md
func TestGo2LeanRepro_StreamedRefusalReachesChatClient(t *testing.T) {
	var r responses.Response
	r.Status = "completed"
	var item responses.ResponseOutputItemUnion
	item.Type, item.ID = "message", "msg_1"
	var part responses.ResponseOutputMessageContentUnion
	part.Type, part.Refusal = "refusal", "I can't help with that."
	item.Content = append(item.Content, part)
	r.Output = append(r.Output, item)

	// What a non-streaming Chat client reads.
	want := nonstream.ConvertResponsesToOpenAIChat(&r, "m").Choices[0].Message.Refusal

	// The same answer as the Responses API streams it.
	c := newResponsesToChatConverter(nil, "m", true)
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.created", Response: r})
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.added", Item: item})
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.refusal.delta", ItemID: item.ID, Delta: part.Refusal})
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.output_item.done", Item: item})
	c.processEvent(&responses.ResponseStreamEventUnion{Type: "response.completed", Response: r})
	var got strings.Builder
	for _, ch := range reproChunks(c.pending) {
		for _, choice := range ch.Choices {
			got.WriteString(choice.Delta.Content)
		}
	}
	if want != "" && !strings.Contains(got.String(), want) {
		t.Errorf("non-streaming refusal %q; the stream delivers content %q and no refusal", want, got.String())
	}
}
