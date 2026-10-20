package nonstream

// Reproductions on the non-streaming converters for findings 59-60, found by
// the go2lean M2 agent with its own specs (a Claude agent, Opus 5.5, that had
// never seen the findings wrote the specs from the package source and
// go2lean's agent prompt; go2lean searched them and `go test` confirmed the
// counterexamples). The tests were written afterwards, with realistic
// JSON-parsed responses. Each test asserts the behaviour a client expects:
// it fails while the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"google.golang.org/genai"
)

// findings/gemini-anthropic-safety-content-filter.md: Gemini blocks the
// answer for safety; the client speaks the Anthropic v1 Messages API.
func TestGo2LeanRepro_GeminiSafetyIsRefusalForAnthropicV1(t *testing.T) {
	var g genai.GenerateContentResponse
	// A blocked candidate carries no content, only the finish reason and the
	// rating that blocked it.
	if err := json.Unmarshal([]byte(`{"candidates":[{"finishReason":"SAFETY","index":0,
	 "safetyRatings":[{"category":"HARM_CATEGORY_DANGEROUS_CONTENT","probability":"HIGH","blocked":true}]}],
	 "usageMetadata":{"promptTokenCount":12,"totalTokenCount":12},"modelVersion":"gemini-2.5-flash"}`), &g); err != nil {
		t.Fatal(err)
	}
	if len(g.Candidates) != 1 || g.Candidates[0].FinishReason != genai.FinishReasonSafety {
		t.Fatalf("fixture not parsed: %+v", g.Candidates)
	}
	beta := HandleGoogleToAnthropicBeta(&g, "m")
	v1 := HandleGoogleToAnthropic(&g, "m")
	if v1.StopReason != anthropic.BetaStopReasonRefusal {
		t.Errorf("Anthropic v1 client: Gemini SAFETY becomes stop_reason %q, not an Anthropic stop reason (the beta path gives %q); clients that check for \"refusal\" miss the block",
			v1.StopReason, beta.StopReason)
	}
}

// findings/gemini-client-responses-coverage.md: a Claude answer with
// extended thinking and a cached prompt, converted for a Gemini client.
func TestGo2LeanRepro_AnthropicToGeminiResponseKeepsThinkingAndCache(t *testing.T) {
	var m anthropic.Message
	if err := json.Unmarshal([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5",
	 "content":[{"type":"thinking","thinking":"The user asks for the capital of France.","signature":"EqQBCkYIBxgCKkA"},{"type":"text","text":"Paris."}],
	 "stop_reason":"stop_sequence","stop_sequence":"END",
	 "usage":{"input_tokens":20,"cache_read_input_tokens":3000,"cache_creation_input_tokens":500,"output_tokens":40}}`), &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Content) != 2 || m.Content[0].Thinking == "" || m.Usage.CacheReadInputTokens != 3000 {
		t.Fatalf("fixture not parsed: %s", m.RawJSON())
	}
	g := HandleAnthropicToGoogle(&m)
	b, _ := json.Marshal(g)
	thought := ""
	for _, p := range g.Candidates[0].Content.Parts {
		if p != nil && p.Thought {
			thought += p.Text
		}
	}
	if thought != m.Content[0].Thinking {
		t.Errorf("thinking block not carried as a thought part: %s", b)
	}
	if fr := g.Candidates[0].FinishReason; fr != genai.FinishReasonStop {
		t.Errorf("stop_sequence becomes finishReason %q, want STOP", fr)
	}
	if u := g.UsageMetadata; u.PromptTokenCount != 3520 || u.CachedContentTokenCount != 3000 {
		t.Errorf("promptTokenCount %d, cachedContentTokenCount %d for 20 uncached + 3000 cache read + 500 cache write input tokens, want 3520 and 3000",
			u.PromptTokenCount, u.CachedContentTokenCount)
	}
}

// findings/gemini-client-responses-coverage.md: an OpenAI refusal converted
// for a Gemini client.
func TestGo2LeanRepro_ChatRefusalReachesGeminiClient(t *testing.T) {
	var c openai.ChatCompletion
	if err := json.Unmarshal([]byte(`{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"gpt-4o","choices":[{"index":0,
	 "message":{"role":"assistant","content":null,"refusal":"I'm sorry, I can't help with that."},"logprobs":null,"finish_reason":"stop"}],
	 "usage":{"prompt_tokens":10,"completion_tokens":9,"total_tokens":19}}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Choices[0].Message.Refusal == "" {
		t.Fatalf("fixture not parsed: %s", c.RawJSON())
	}
	g := HandleOpenAIToGoogle(&c)
	text := ""
	for _, p := range g.Candidates[0].Content.Parts {
		if p != nil {
			text += p.Text
		}
	}
	if text != c.Choices[0].Message.Refusal {
		b, _ := json.Marshal(g)
		t.Errorf("the refusal does not reach the Gemini client, which gets an empty answer with finishReason STOP: %s", b)
	}
}
