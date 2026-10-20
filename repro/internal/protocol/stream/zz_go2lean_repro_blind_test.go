package stream

// Reproduction on the Chat -> Anthropic stream converters for finding 48
// (chat-stop-with-tool-calls-end-turn), found by go2lean from a spec written
// blind on the non-stream converter (a Claude agent that had never seen the
// findings wrote it from the package source and go2lean's agent prompt);
// the stream mappers have the same mapping, found by reading and confirmed
// here. The test asserts the behaviour a client expects: it fails while the
// bug is there.

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

func reproBlChatStreamStopReason(t *testing.T, mapper func(string) string, chunks ...string) (stopReason string, sawToolUse bool) {
	t.Helper()
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString("data: " + c + "\n\n")
	}
	b.WriteString("data: [DONE]\n\n")
	resp := &http.Response{Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(b.String()))}
	s := ssestream.NewStream[openai.ChatCompletionChunk](ssestream.NewDecoder(resp), nil)
	conv := newOpenAIToAnthropicConverter(s, "m", nil, nil, mapper)
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			return
		}
		ae, ok := AsAnthropicEvent(ev)
		if !ok {
			continue
		}
		var m struct {
			ContentBlock struct {
				Type string `json:"type"`
			} `json:"content_block"`
			Delta struct {
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(ae.RawJSON()), &m); err != nil {
			t.Fatal(err)
		}
		switch ae.Type {
		case "content_block_start":
			if m.ContentBlock.Type == "tool_use" {
				sawToolUse = true
			}
		case "message_delta":
			stopReason = m.Delta.StopReason
		}
	}
}

func TestGo2LeanRepro_StreamChatStopWithToolCallsIsToolUse(t *testing.T) {
	chunk := func(delta, finish string) string {
		return `{"id":"c","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":` +
			delta + `,"logprobs":null,"finish_reason":` + finish + `}]}`
	}
	chunks := []string{
		chunk(`{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]}`, "null"),
		chunk(`{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":\"Paris\"}"}}]}`, "null"),
		chunk(`{}`, `"stop"`),
	}
	for name, mapper := range map[string]func(string) string{
		"v1": mapOpenAIFinishReasonToAnthropic, "beta": mapOpenAIFinishReasonToAnthropicBeta,
	} {
		stop, sawToolUse := reproBlChatStreamStopReason(t, mapper, chunks...)
		if !sawToolUse {
			t.Fatalf("%s: no tool_use block streamed", name)
		}
		if stop != "tool_use" {
			t.Errorf("%s: a stream with a tool_use block ends with stop_reason %q; clients that run tools on \"tool_use\" end the turn instead", name, stop)
		}
	}
}
