package request

// Reproductions on the request conversions for findings 44-56, found by
// go2lean from specs written blind (a Claude agent that had never seen the
// findings wrote them from the package source and go2lean's agent prompt;
// go2lean searched them and `go test` confirmed the counterexamples), except
// findings/anthropic-gemini-image-double-encoded.md, found by hand while
// triaging a spurious counterexample of that run. The tests were written
// afterwards, with realistic JSON-parsed requests. Each test asserts the
// behaviour a client expects: it fails while the bug is there.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

func reproBlJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func reproBlBeta(t *testing.T, body string) *anthropic.BetaMessageNewParams {
	t.Helper()
	var req anthropic.BetaMessageNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func reproBlV1(t *testing.T, body string) *anthropic.MessageNewParams {
	t.Helper()
	var req anthropic.MessageNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func reproBlChat(t *testing.T, body string) *openai.ChatCompletionNewParams {
	t.Helper()
	var req openai.ChatCompletionNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func reproBlRaw(r gjson.Result) string {
	if !r.Exists() {
		return "(absent)"
	}
	return r.Raw
}

const reproBlWeatherTool = `{"name":"get_weather","description":"Current weather for a city",
 "input_schema":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}`

// findings/anthropic-gemini-tool-use-args.md: the second request of a tool
// loop resends the assistant's tool_use with its input.
const reproBlToolLoopBody = `{"model":"m","max_tokens":1024,"tools":[` + reproBlWeatherTool + `],
"messages":[
 {"role":"user","content":"Weather in Paris?"},
 {"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Paris"}}]},
 {"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"18C, sunny"}]}]}`

func reproBlFunctionCallArgs(contents []*genai.Content) (map[string]any, bool) {
	for _, c := range contents {
		for _, p := range c.Parts {
			if p.FunctionCall != nil {
				return p.FunctionCall.Args, true
			}
		}
	}
	return nil, false
}

func TestGo2LeanRepro_AnthropicToGeminiKeepsToolUseArgs(t *testing.T) {
	_, beta, _ := ConvertAnthropicBetaToGoogleRequest(reproBlBeta(t, reproBlToolLoopBody), 4096)
	_, v1, _ := ConvertAnthropicToGoogleRequest(reproBlV1(t, reproBlToolLoopBody), 4096)
	for name, contents := range map[string][]*genai.Content{"beta": beta, "v1": v1} {
		args, ok := reproBlFunctionCallArgs(contents)
		if !ok {
			t.Fatalf("%s: no functionCall part: %s", name, reproBlJSON(t, contents))
		}
		if args["city"] != "Paris" {
			t.Errorf("%s: tool_use input {\"city\":\"Paris\"} becomes functionCall args %s; Gemini sees the call without its arguments", name, reproBlJSON(t, args))
		}
	}
}

// findings/anthropic-tool-choice-none-responses-gemini.md: a custom tool and
// tool_choice none, to force a text-only turn.
const reproBlToolChoiceNoneBody = `{"model":"m","max_tokens":1024,"tools":[` + reproBlWeatherTool + `],
"tool_choice":{"type":"none"},
"messages":[{"role":"user","content":"Summarize the conversation so far."}]}`

func TestGo2LeanRepro_AnthropicToolChoiceNoneReachesResponses(t *testing.T) {
	beta := reproBlJSON(t, ConvertAnthropicBetaToResponsesRequest(reproBlBeta(t, reproBlToolChoiceNoneBody)))
	v1 := reproBlJSON(t, ConvertAnthropicV1ToResponsesRequest(reproBlV1(t, reproBlToolChoiceNoneBody)))
	for name, w := range map[string]string{"beta": beta, "v1": v1} {
		if !gjson.Get(w, "tools.0").Exists() {
			t.Fatalf("%s: tools not carried: %s", name, w)
		}
		if tc := gjson.Get(w, "tool_choice").Raw; tc != `"none"` {
			t.Errorf("%s: tool_choice none becomes Responses tool_choice %s with the tools still declared, want \"none\"", name, tc)
		}
	}
}

func TestGo2LeanRepro_AnthropicToolChoiceNoneReachesGemini(t *testing.T) {
	_, _, cfg := ConvertAnthropicBetaToGoogleRequest(reproBlBeta(t, reproBlToolChoiceNoneBody), 4096)
	if len(cfg.Tools) == 0 || len(cfg.Tools[0].FunctionDeclarations) == 0 {
		t.Fatalf("tools not carried: %s", reproBlJSON(t, cfg))
	}
	if cfg.ToolConfig == nil || cfg.ToolConfig.FunctionCallingConfig == nil ||
		cfg.ToolConfig.FunctionCallingConfig.Mode != genai.FunctionCallingConfigModeNone {
		t.Errorf("tool_choice none becomes toolConfig %s with the function still declared (Gemini's default is AUTO), want mode NONE",
			reproBlJSON(t, cfg.ToolConfig))
	}
}

// findings/chat-responses-developer-dropped.md: current OpenAI SDKs send
// the system prompt as a developer message.
func TestGo2LeanRepro_ChatDeveloperMessageReachesResponses(t *testing.T) {
	req := reproBlChat(t, `{"model":"gpt-5","messages":[
	 {"role":"developer","content":"Answer in French."},
	 {"role":"user","content":"What time is it?"}]}`)
	w := reproBlJSON(t, ConvertChatToOpenAIResponses(req, 0))
	if !strings.Contains(w, "Answer in French.") {
		t.Errorf("developer message dropped (in neither instructions nor input): instructions %s, input %s",
			reproBlRaw(gjson.Get(w, "instructions")), gjson.Get(w, "input").Raw)
	}
}

// findings/chat-gemini-request-coverage.md: three requests a current OpenAI
// SDK client sends.
func TestGo2LeanRepro_ChatToGeminiKeepsMaxCompletionTokens(t *testing.T) {
	req := reproBlChat(t, `{"model":"gemini-2.5-pro","max_completion_tokens":16000,
	 "messages":[{"role":"user","content":"Write the full report."}]}`)
	_, _, cfg := ConvertOpenAIToGoogleRequest(req, 4096)
	if cfg.MaxOutputTokens != 16000 {
		t.Errorf("max_completion_tokens 16000 becomes maxOutputTokens %d (the gateway default)", cfg.MaxOutputTokens)
	}
}

func TestGo2LeanRepro_ChatToGeminiKeepsDeveloperAndSystemText(t *testing.T) {
	req := reproBlChat(t, `{"model":"gemini-2.5-pro","messages":[
	 {"role":"system","content":[{"type":"text","text":"You are a travel agent."}]},
	 {"role":"developer","content":"Answer in French."},
	 {"role":"user","content":"Plan a weekend in Rome."}]}`)
	_, _, cfg := ConvertOpenAIToGoogleRequest(req, 4096)
	sys := reproBlJSON(t, cfg.SystemInstruction)
	for _, want := range []string{"You are a travel agent.", "Answer in French."} {
		if !strings.Contains(sys, want) {
			t.Errorf("%q missing from the systemInstruction %s", want, sys)
		}
	}
}

func TestGo2LeanRepro_ChatToGeminiKeepsUserImages(t *testing.T) {
	req := reproBlChat(t, `{"model":"gemini-2.5-pro","messages":[{"role":"user","content":[
	 {"type":"text","text":"What is in this picture?"},
	 {"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgo="}}]}]}`)
	_, contents, _ := ConvertOpenAIToGoogleRequest(req, 4096)
	found := false
	for _, c := range contents {
		for _, p := range c.Parts {
			if p.InlineData != nil || p.FileData != nil {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("user image_url part dropped, only the text is sent: %s", reproBlJSON(t, contents))
	}
}

// findings/anthropic-disable-parallel-tool-use.md: an agent loop that runs
// one tool at a time.
func TestGo2LeanRepro_DisableParallelToolUseCarried(t *testing.T) {
	body := `{"model":"m","max_tokens":1024,"tools":[` + reproBlWeatherTool + `],
	"tool_choice":{"type":"auto","disable_parallel_tool_use":true},
	"messages":[{"role":"user","content":"Weather in Paris and Rome?"}]}`
	chat, _ := ConvertAnthropicBetaToOpenAIRequest(reproBlBeta(t, body), false, false, false)
	if p := gjson.Get(reproBlJSON(t, chat), "parallel_tool_calls"); !p.Exists() || p.Bool() {
		t.Errorf("Anthropic -> Chat: disable_parallel_tool_use:true gives parallel_tool_calls %s, want false", reproBlRaw(p))
	}
	resp := ConvertAnthropicBetaToResponsesRequest(reproBlBeta(t, body))
	if p := gjson.Get(reproBlJSON(t, resp), "parallel_tool_calls"); !p.Exists() || p.Bool() {
		t.Errorf("Anthropic -> Responses: disable_parallel_tool_use:true gives parallel_tool_calls %s, want false", reproBlRaw(p))
	}
}

// findings/anthropic-chat-tool-strict.md: strict tool use, in the three
// directions finding 30 does not cover.
func TestGo2LeanRepro_AnthropicChatToolStrictCarried(t *testing.T) {
	body := `{"model":"m","max_tokens":1024,"tools":[{"name":"get_weather","strict":true,
	 "input_schema":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"],"additionalProperties":false}}],
	"messages":[{"role":"user","content":"Weather in Paris?"}]}`
	chat, _ := ConvertAnthropicBetaToOpenAIRequest(reproBlBeta(t, body), false, false, false)
	if s := gjson.Get(reproBlJSON(t, chat), "tools.0.function.strict"); !s.Bool() {
		t.Errorf("Anthropic -> Chat: strict:true not carried: %s", gjson.Get(reproBlJSON(t, chat), "tools").Raw)
	}
	resp := reproBlJSON(t, ConvertAnthropicBetaToResponsesRequest(reproBlBeta(t, body)))
	if s := gjson.Get(resp, "tools.0.strict"); !s.Bool() {
		t.Errorf("Anthropic -> Responses: strict:true not carried: %s", gjson.Get(resp, "tools").Raw)
	}
	creq := reproBlChat(t, `{"model":"m","messages":[{"role":"user","content":"Weather in Paris?"}],
	"tools":[{"type":"function","function":{"name":"get_weather","strict":true,
	 "parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"],"additionalProperties":false}}}]}`)
	anth := reproBlJSON(t, ConvertOpenAIToAnthropicRequest(creq, 1024))
	if s := gjson.Get(anth, "tools.0.strict"); !s.Bool() {
		t.Errorf("Chat -> Anthropic: strict:true not carried: %s", gjson.Get(anth, "tools").Raw)
	}
}

// findings/chat-responses-request-flags.md: request flags a Chat client
// sets for attribution, storage and pricing.
func TestGo2LeanRepro_ChatToResponsesKeepsRequestFlags(t *testing.T) {
	req := reproBlChat(t, `{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],
	 "user":"user-42","safety_identifier":"sid-42","metadata":{"session":"s1"},
	 "service_tier":"flex","store":false,"logprobs":true,"top_logprobs":3}`)
	w := reproBlJSON(t, ConvertChatToOpenAIResponses(req, 0))
	var missing []string
	for _, k := range []string{"user", "safety_identifier", "metadata.session", "service_tier", "store", "top_logprobs"} {
		if !gjson.Get(w, k).Exists() {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		t.Errorf("Chat -> Responses drops %s (Responses stores the response unless store is false): %s", strings.Join(missing, ", "), w)
	}
}

// findings/gemini-anthropic-sampling-dropped.md (latent: no gateway
// caller): a Gemini request with sampling parameters and a stop sequence.
func TestGo2LeanRepro_GeminiToAnthropicKeepsSampling(t *testing.T) {
	temp, topP, topK := float32(0.2), float32(0.9), float32(40)
	cfg := &genai.GenerateContentConfig{
		MaxOutputTokens: 1024,
		Temperature:     &temp,
		TopP:            &topP,
		TopK:            &topK,
		StopSequences:   []string{"END"},
	}
	contents := []*genai.Content{genai.NewContentFromText("List three colours, then write END.", genai.RoleUser)}
	w := reproBlJSON(t, ConvertGoogleToAnthropicRequest("m", contents, cfg))
	var missing []string
	for _, k := range []string{"temperature", "top_p", "top_k", "stop_sequences"} {
		if !gjson.Get(w, k).Exists() {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		t.Errorf("Gemini -> Anthropic drops %s: %s", strings.Join(missing, ", "), w)
	}
}

// findings/anthropic-gemini-image-double-encoded.md (found by hand): a
// pasted screenshot, as Claude Code sends it.
func TestGo2LeanRepro_AnthropicToGeminiImageBytesDecoded(t *testing.T) {
	const b64 = "iVBORw0KGgo="
	body := `{"model":"m","max_tokens":1024,"messages":[{"role":"user","content":[
	 {"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + b64 + `"}},
	 {"type":"text","text":"What is in this screenshot?"}]}]}`
	_, contents, _ := ConvertAnthropicBetaToGoogleRequest(reproBlBeta(t, body), 4096)
	var blob *genai.Blob
	for _, c := range contents {
		for _, p := range c.Parts {
			if p.InlineData != nil {
				blob = p.InlineData
			}
		}
	}
	if blob == nil {
		t.Fatalf("image dropped: %s", reproBlJSON(t, contents))
	}
	want := "\x89PNG\r\n\x1a\n"
	if string(blob.Data) != want {
		t.Errorf("inlineData holds the base64 text %q instead of the image bytes %q; on the wire Gemini gets data %s, the base64 encoded twice",
			blob.Data, want, gjson.Get(reproBlJSON(t, blob), "data").Raw)
	}
}
