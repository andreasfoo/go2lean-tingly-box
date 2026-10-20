package request

// Reproductions on the request conversions for findings 57-58, found by the
// go2lean M2 agent with its own specs (a Claude agent, Opus 5.5, that had
// never seen the findings wrote the specs from the package source and
// go2lean's agent prompt; go2lean searched them and `go test` confirmed the
// counterexamples). The tests were written afterwards, with realistic
// JSON-parsed requests. Each test asserts the behaviour a client expects: it
// fails while the bug is there.

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

// findings/gemini-max-tokens-int32-wrap.md: a client that sends a huge
// max_tokens to mean "no limit", routed to a Google-style provider.
func TestGo2LeanRepro_GeminiMaxTokensDoesNotWrap(t *testing.T) {
	for _, n := range []int64{4294967396, 9007199254740991} { // 2^32+100, Number.MAX_SAFE_INTEGER
		body := `{"model":"gemini-2.5-pro","max_tokens":` + reproM2JSONInt(n) + `,"messages":[{"role":"user","content":"Write a long story."}]}`
		var v1 anthropic.MessageNewParams
		var beta anthropic.BetaMessageNewParams
		var chat openai.ChatCompletionNewParams
		for _, p := range []any{&v1, &beta, &chat} {
			if err := json.Unmarshal([]byte(body), p); err != nil {
				t.Fatal(err)
			}
		}
		if v1.MaxTokens != n || beta.MaxTokens != n || chat.MaxTokens.Value != n {
			t.Fatalf("fixture not parsed: %d %d %d", v1.MaxTokens, beta.MaxTokens, chat.MaxTokens.Value)
		}
		configs := map[string]*genai.GenerateContentConfig{}
		_, _, configs["Anthropic v1"] = ConvertAnthropicToGoogleRequest(&v1, 4096)
		_, _, configs["Anthropic beta"] = ConvertAnthropicBetaToGoogleRequest(&beta, 4096)
		_, _, configs["Chat"] = ConvertOpenAIToGoogleRequest(&chat, 4096)
		for name, cfg := range configs {
			got := int64(cfg.MaxOutputTokens)
			if got <= 0 || got < min(n, math.MaxInt32) && got == int64(int32(n)) {
				t.Errorf("%s client: max_tokens %d becomes maxOutputTokens %d (wrapped to int32); Gemini cuts the answer short or rejects a non-positive limit",
					name, n, got)
			}
		}
	}
}

func reproM2JSONInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// findings/responses-chat-parallel-tool-calls.md: a Responses agent that runs
// tools one at a time, routed to a Chat provider.
func TestGo2LeanRepro_ResponsesToChatKeepsParallelToolCalls(t *testing.T) {
	var params responses.ResponseNewParams
	if err := json.Unmarshal([]byte(`{"model":"gpt-4.1","input":"What is the weather in Paris and Rome?","parallel_tool_calls":false,
	 "tools":[{"type":"function","name":"get_weather","description":"Current weather for a city","parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]},"strict":false}]}`), &params); err != nil {
		t.Fatal(err)
	}
	if !params.ParallelToolCalls.Valid() || params.ParallelToolCalls.Value {
		t.Fatalf("fixture not parsed: parallel_tool_calls %+v", params.ParallelToolCalls)
	}
	chat := ConvertOpenAIResponsesToChat(&params, 4096)
	b, err := json.Marshal(chat)
	if err != nil {
		t.Fatal(err)
	}
	if r := gjson.GetBytes(b, "parallel_tool_calls"); !r.Exists() || r.Bool() {
		got := r.Raw
		if !r.Exists() {
			got = "(absent)"
		}
		t.Errorf("parallel_tool_calls false not carried to the Chat request (got %s); the provider may return several tool calls in one turn: %s", got, b)
	}
}
