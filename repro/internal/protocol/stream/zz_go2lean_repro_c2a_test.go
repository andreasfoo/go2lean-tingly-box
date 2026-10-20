package stream

// Reproductions on the Chat -> Anthropic stream converter, from the bug
// classes of similar gateways (hunting-similar-tools.md). Each test asserts
// the behaviour a client expects: it fails while the bug is there.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

// reproChatToAnthropic feeds Chat chunk JSON through the Chat -> Anthropic
// stream converter and returns the emitted events as (type, payload).
func reproChatToAnthropic(t *testing.T, chunks ...string) (types []string, payloads []map[string]any) {
	t.Helper()
	var b strings.Builder
	for _, c := range chunks {
		b.WriteString("data: " + c + "\n\n")
	}
	b.WriteString("data: [DONE]\n\n")
	resp := &http.Response{Header: http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(b.String()))}
	s := ssestream.NewStream[openai.ChatCompletionChunk](ssestream.NewDecoder(resp), nil)
	conv := newOpenAIToAnthropicConverter(s, "m", nil, nil, mapOpenAIFinishReasonToAnthropic)
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
		var m map[string]any
		if err := json.Unmarshal([]byte(ae.RawJSON()), &m); err != nil {
			t.Fatal(err)
		}
		types = append(types, ae.Type)
		payloads = append(payloads, m)
	}
}

func reproDeepSeekChunk(delta, finish string) string {
	return `{"id":"c","object":"chat.completion.chunk","created":1,"model":"deepseek-reasoner","choices":[{"index":0,"delta":` +
		delta + `,"logprobs":null,"finish_reason":` + finish + `}]}`
}

// findings/chat-anthropic-reasoning-null.md: DeepSeek's reasoning models
// stream the answer as {"content":"…","reasoning_content":null}. A null
// reasoning_content after the answer started opens a second, empty
// thinking block while the text block is still open.
func TestGo2LeanRepro_NullReasoningOpensNoThinkingBlock(t *testing.T) {
	types, payloads := reproChatToAnthropic(t,
		reproDeepSeekChunk(`{"role":"assistant","content":null,"reasoning_content":"Think."}`, "null"),
		reproDeepSeekChunk(`{"content":"It","reasoning_content":null}`, "null"),
		reproDeepSeekChunk(`{"content":" works.","reasoning_content":null}`, "null"),
		reproDeepSeekChunk(`{"content":"","reasoning_content":null}`, `"stop"`),
	)
	var blocks []string
	open := -1
	var problems []string
	for i, ty := range types {
		switch ty {
		case "content_block_start":
			idx := int(payloads[i]["index"].(float64))
			blocks = append(blocks, payloads[i]["content_block"].(map[string]any)["type"].(string))
			if open != -1 {
				problems = append(problems, fmt.Sprintf("block %d starts while block %d is open", idx, open))
			}
			open = idx
		case "content_block_stop":
			open = -1
		}
	}
	if got := strings.Join(blocks, ","); got != "thinking,text" || len(problems) > 0 {
		t.Errorf("content blocks [%s], want [thinking,text]; %v", got, problems)
	}
}
