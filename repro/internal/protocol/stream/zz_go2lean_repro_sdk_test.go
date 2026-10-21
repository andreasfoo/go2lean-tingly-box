package stream

// Reproductions checked with the clients' own SDKs as the oracle: what a
// client's SDK makes of tingly-box's converted stream must be what the
// provider said. Each test fails while the bug is there.

import (
	"bytes"
	"encoding/json"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicstream "github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/v3"
	openaistream "github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"
	"google.golang.org/genai"

	"github.com/tingly-dev/tingly-box/internal/protocol"
	"github.com/tingly-dev/tingly-box/internal/protocol/request"
)

// geminiTextThenTool is a Gemini answer that says something, then calls a
// tool: the usual shape of an agent turn.
func geminiTextThenTool(yield func(*genai.GenerateContentResponse, error) bool) {
	if !yield(&genai.GenerateContentResponse{Candidates: []*genai.Candidate{{Content: &genai.Content{Role: "model",
		Parts: []*genai.Part{genai.NewPartFromText("Let me check.")}}}}}, nil) {
		return
	}
	yield(&genai.GenerateContentResponse{Candidates: []*genai.Candidate{{Content: &genai.Content{Role: "model",
		Parts: []*genai.Part{{FunctionCall: &genai.FunctionCall{ID: "call_1", Name: "get_weather", Args: map[string]any{"city": "Paris"}}}}},
		FinishReason: genai.FinishReasonStop}},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 5, TotalTokenCount: 15}}, nil)
}

// sseData returns the data payloads of an SSE body.
func sseData(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if data, ok := strings.CutPrefix(line, "data:"); ok {
			out = append(out, strings.TrimSpace(data))
		}
	}
	return out
}

func TestGo2LeanRepro_GeminiToAnthropicStreamAccumulates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, beta := range []bool{false, true} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", nil)
		var err error
		if beta {
			_, err = HandleGoogleToAnthropicBetaStreamResponse(c, geminiTextThenTool, "gemini-2.5-pro")
		} else {
			_, err = HandleGoogleToAnthropicStreamResponse(c, geminiTextThenTool, "gemini-2.5-pro")
		}
		if err != nil {
			t.Fatal(err)
		}
		name := map[bool]string{false: "v1", true: "beta"}[beta]
		// The client's view: anthropic-sdk-go's Message.Accumulate.
		var msg anthropic.Message
		for _, data := range sseData(w.Body.String()) {
			var ev anthropic.MessageStreamEventUnion
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				t.Fatalf("%s: event does not decode: %v", name, err)
			}
			if err := msg.Accumulate(ev); err != nil {
				t.Errorf("%s: the Anthropic SDK rejects the stream: %v", name, err)
				break
			}
		}
		var types []string
		for _, b := range msg.Content {
			types = append(types, b.Type)
		}
		if len(types) != 2 || types[0] != "text" || types[1] != "tool_use" {
			t.Errorf("%s: the client sees content blocks %v; want [text tool_use] (the tool call is lost)", name, types)
		}
	}
}

// ---- harness for findings 63-74 (helpers prefixed sdkeq: SDK as oracle) ----

type sdkeqRecorder struct{ *httptest.ResponseRecorder }

func (r *sdkeqRecorder) CloseNotify() <-chan bool { return make(chan bool) }

func sdkeqGin() (*gin.Context, *sdkeqRecorder) {
	gin.SetMode(gin.TestMode)
	w := &sdkeqRecorder{httptest.NewRecorder()}
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	return c, w
}

// sdkeqData returns the data payloads of an SSE body (without [DONE]).
func sdkeqData(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if data, ok := strings.CutPrefix(line, "data:"); ok {
			data = strings.TrimSpace(data)
			if data != "[DONE]" {
				out = append(out, data)
			}
		}
	}
	return out
}

// sdkeqSSE frames JSON payloads as SSE; with typed=true each frame also
// carries an `event:` line taken from the payload's "type".
func sdkeqSSE(typed bool, payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		if typed {
			var head struct{ Type string }
			_ = json.Unmarshal([]byte(p), &head)
			b.WriteString("event: " + head.Type + "\n")
		}
		b.WriteString("data: " + p + "\n\n")
	}
	if !typed {
		b.WriteString("data: [DONE]\n\n")
	}
	return b.String()
}

func sdkeqHTTP(sse string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(sse)),
	}
}

func sdkeqChatUpstream(chunks ...string) *openaistream.Stream[openai.ChatCompletionChunk] {
	return openaistream.NewStream[openai.ChatCompletionChunk](openaistream.NewDecoder(sdkeqHTTP(sdkeqSSE(false, chunks...))), nil)
}

func sdkeqResponsesUpstream(events ...string) *openaistream.Stream[responses.ResponseStreamEventUnion] {
	return openaistream.NewStream[responses.ResponseStreamEventUnion](openaistream.NewDecoder(sdkeqHTTP(sdkeqSSE(true, events...))), nil)
}

func sdkeqAnthropicUpstream(events ...string) *anthropicstream.Stream[anthropic.BetaRawMessageStreamEventUnion] {
	return anthropicstream.NewStream[anthropic.BetaRawMessageStreamEventUnion](anthropicstream.NewDecoder(sdkeqHTTP(sdkeqSSE(true, events...))), nil)
}

// sdkeqGemini decodes Gemini wire JSON the way go-genai does for a client.
func sdkeqGemini(t *testing.T, js ...string) []*genai.GenerateContentResponse {
	t.Helper()
	var out []*genai.GenerateContentResponse
	for _, j := range js {
		var r genai.GenerateContentResponse
		if err := json.Unmarshal([]byte(j), &r); err != nil {
			t.Fatalf("gemini fixture: %v", err)
		}
		out = append(out, &r)
	}
	return out
}

func sdkeqSeq(rs []*genai.GenerateContentResponse) iter.Seq2[*genai.GenerateContentResponse, error] {
	return func(yield func(*genai.GenerateContentResponse, error) bool) {
		for _, r := range rs {
			if !yield(r, nil) {
				return
			}
		}
	}
}

// sdkeqChatClient feeds a Chat SSE body to openai-go's ChatCompletionAccumulator.
func sdkeqChatClient(t *testing.T, body string) openai.ChatCompletionAccumulator {
	t.Helper()
	var acc openai.ChatCompletionAccumulator
	for _, data := range sdkeqData(body) {
		if strings.HasPrefix(data, `{"error"`) {
			continue
		}
		var chunk openai.ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			t.Fatalf("chunk does not decode: %v: %s", err, data)
		}
		if !acc.AddChunk(chunk) {
			t.Errorf("openai-go's ChatCompletionAccumulator rejects chunk %s", data)
		}
	}
	return acc
}

// sdkeqAnthropicClient feeds Anthropic SSE payloads to Message.Accumulate.
func sdkeqAnthropicClient(t *testing.T, payloads []string) anthropic.Message {
	t.Helper()
	var msg anthropic.Message
	for _, data := range payloads {
		var ev anthropic.MessageStreamEventUnion
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("event does not decode: %v: %s", err, data)
		}
		if err := msg.Accumulate(ev); err != nil {
			t.Fatalf("anthropic-sdk-go rejects the stream: %v", err)
		}
	}
	return msg
}

// sdkeqGeminiToAnthropic runs both Gemini -> Anthropic stream handlers and
// returns what the client's SDK accumulates from each.
func sdkeqGeminiToAnthropic(t *testing.T, rs []*genai.GenerateContentResponse) map[string]anthropic.Message {
	t.Helper()
	out := map[string]anthropic.Message{}
	for _, beta := range []bool{false, true} {
		c, w := sdkeqGin()
		var err error
		if beta {
			_, err = HandleGoogleToAnthropicBetaStreamResponse(c, sdkeqSeq(rs), "gemini-2.5-pro")
		} else {
			_, err = HandleGoogleToAnthropicStreamResponse(c, sdkeqSeq(rs), "gemini-2.5-pro")
		}
		if err != nil {
			t.Fatal(err)
		}
		out[map[bool]string{false: "v1", true: "beta"}[beta]] = sdkeqAnthropicClient(t, sdkeqData(w.Body.String()))
	}
	return out
}

func sdkeqGeminiToChat(t *testing.T, rs []*genai.GenerateContentResponse) openai.ChatCompletionAccumulator {
	t.Helper()
	c, w := sdkeqGin()
	if err := HandleGoogleToOpenAIStreamResponse(c, sdkeqSeq(rs), "gemini-2.5-pro"); err != nil {
		t.Fatal(err)
	}
	return sdkeqChatClient(t, w.Body.String())
}

func sdkeqIsJSONObject(raw string) bool {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return false
	}
	_, ok := v.(map[string]any)
	return ok
}

// sdkeqResponsesCompleted returns the response of the terminal event a
// Responses client receives.
func sdkeqResponsesCompleted(t *testing.T, body string) responses.Response {
	t.Helper()
	for _, data := range sdkeqData(body) {
		var ev responses.ResponseStreamEventUnion
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Type == "response.completed" || ev.Type == "response.incomplete" {
			return ev.Response
		}
	}
	t.Fatal("no terminal response event")
	return responses.Response{}
}

func sdkeqTypes(resp responses.Response) []string {
	var out []string
	for _, item := range resp.Output {
		out = append(out, item.Type)
	}
	return out
}

// sdkeqChatClientErr reads a Chat SSE body with openai-go's stream and
// returns the error the client gets.
func sdkeqChatClientErr(body string) error {
	s := openaistream.NewStream[openai.ChatCompletionChunk](openaistream.NewDecoder(sdkeqHTTP(body)), nil)
	for s.Next() {
	}
	return s.Err()
}

// ---- Finding 63: Gemini thoughtSignature dropped ---------------------------

func TestGo2LeanRepro_GeminiThoughtSignatureRoundTrips(t *testing.T) {
	sig := []byte("gemini-thought-signature")
	rs := sdkeqGemini(t, `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"`+
		"Z2VtaW5pLXRob3VnaHQtc2lnbmF0dXJl"+`"}]},"finishReason":"STOP"}]}`)
	if got := rs[0].Candidates[0].Content.Parts[0].ThoughtSignature; !bytes.Equal(got, sig) {
		t.Fatalf("fixture: signature %q", got)
	}
	// A go-genai client would resend the model content as is, signature included.
	check := func(name string, contents []*genai.Content) {
		for _, content := range contents {
			for _, p := range content.Parts {
				if p.FunctionCall != nil && p.FunctionCall.Name == "get_weather" {
					if !bytes.Equal(p.ThoughtSignature, sig) {
						t.Errorf("%s: next Gemini request's functionCall part has thoughtSignature %q; want %q, as Gemini sent it (Gemini 3 rejects the turn)", name, p.ThoughtSignature, sig)
					}
					return
				}
			}
		}
		t.Errorf("%s: next Gemini request has no functionCall part", name)
	}

	// Anthropic client (Claude Code): streamed answer -> Accumulate -> ToParam -> next request.
	for name, msg := range sdkeqGeminiToAnthropic(t, rs) {
		var toolID string
		for _, b := range msg.Content {
			if b.Type == "tool_use" {
				toolID = b.ID
			}
		}
		next := anthropic.MessageNewParams{
			Model: "gemini-3-pro-preview", MaxTokens: 1024,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock("Weather in Paris?")),
				msg.ToParam(),
				anthropic.NewUserMessage(anthropic.NewToolResultBlock(toolID, "sunny", false)),
			},
		}
		_, contents, _ := request.ConvertAnthropicToGoogleRequest(&next, 1024)
		check("anthropic "+name, contents)
	}

	// Chat client: streamed answer -> accumulator -> ToParam -> next request.
	acc := sdkeqGeminiToChat(t, rs)
	if len(acc.Choices) == 0 || len(acc.Choices[0].Message.ToolCalls) == 0 {
		t.Fatalf("chat: no tool call")
	}
	next := openai.ChatCompletionNewParams{
		Model: "gemini-3-pro-preview",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Weather in Paris?"),
			acc.Choices[0].Message.ToParam(),
			openai.ToolMessage("sunny", acc.Choices[0].Message.ToolCalls[0].ID),
		},
	}
	_, contents, _ := request.ConvertOpenAIToGoogleRequest(&next, 1024)
	check("chat", contents)
}

// ---- Finding 64: Gemini -> Chat parallel calls merge -------------------------

func TestGo2LeanRepro_GeminiToChatParallelToolCalls(t *testing.T) {
	rs := sdkeqGemini(t, `{"candidates":[{"content":{"role":"model","parts":[`+
		`{"functionCall":{"id":"call_a","name":"get_weather","args":{"city":"Paris"}}},`+
		`{"functionCall":{"id":"call_b","name":"get_time","args":{"tz":"Europe/Paris"}}}]},"finishReason":"STOP"}],`+
		`"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":8,"totalTokenCount":18}}`)
	var want []string
	for _, r := range rs {
		for _, fc := range r.FunctionCalls() {
			want = append(want, fc.Name)
		}
	}
	acc := sdkeqGeminiToChat(t, rs)
	var got []string
	for _, tc := range acc.Choices[0].Message.ToolCalls {
		got = append(got, tc.Function.Name+" "+tc.Function.Arguments)
	}
	if len(got) != len(want) {
		t.Errorf("chat client (openai-go accumulator) sees tool calls %q; go-genai sees %v", got, want)
	}
}

// ---- Finding 67: argument-less Gemini call, stream ----------------------------

func TestGo2LeanRepro_GeminiArgumentlessCallInputIsObject(t *testing.T) {
	// Gemini functionCall without "args" (a call to a tool with no parameters).
	rs := sdkeqGemini(t, `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"call_1","name":"get_time"}}]},"finishReason":"STOP"}]}`)
	for name, msg := range sdkeqGeminiToAnthropic(t, rs) {
		// The client resends the turn: Message.ToParam, as the SDK's tool loop does.
		raw, err := json.Marshal(msg.ToParam())
		if err != nil {
			t.Fatal(err)
		}
		var param struct {
			Content []struct {
				Type  string          `json:"type"`
				Input json.RawMessage `json:"input"`
			} `json:"content"`
		}
		_ = json.Unmarshal(raw, &param)
		for _, b := range param.Content {
			if b.Type == "tool_use" && !sdkeqIsJSONObject(string(b.Input)) {
				t.Errorf("%s: resent tool_use input %s; want an object such as {} (Anthropic rejects the next request)", name, b.Input)
			}
		}
	}
	acc := sdkeqGeminiToChat(t, rs)
	for _, tc := range acc.Choices[0].Message.ToolCalls {
		if !sdkeqIsJSONObject(tc.Function.Arguments) {
			t.Errorf("chat: tool call arguments %q; want a JSON object such as {}", tc.Function.Arguments)
		}
	}
}

// ---- Findings 68 and 69: Chat -> Responses stream, reasoning and refusal ----

func TestGo2LeanRepro_ChatToResponsesStreamKeepsReasoning(t *testing.T) {
	chunks := []string{
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"deepseek-reasoner","choices":[{"index":0,"delta":{"role":"assistant","content":null,"reasoning_content":"The user greets me."}}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"deepseek-reasoner","choices":[{"index":0,"delta":{"content":"Hello!"}}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"deepseek-reasoner","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":7,"total_tokens":12}}`,
	}
	c, w := sdkeqGin()
	if _, err := HandleOpenAIChatToResponsesStream(protocol.NewHandleContext(c, "m"), sdkeqChatUpstream(chunks...), "m"); err != nil {
		t.Fatal(err)
	}
	resp := sdkeqResponsesCompleted(t, w.Body.String())
	var reasoning string
	for _, item := range resp.Output {
		if item.Type == "reasoning" {
			for _, s := range item.Summary {
				reasoning += s.Text
			}
			for _, ct := range item.Content {
				reasoning += ct.Text
			}
		}
	}
	if reasoning != "The user greets me." {
		t.Errorf("responses client reasoning %q; upstream reasoning_content %q (output item types %v)", reasoning, "The user greets me.", sdkeqTypes(resp))
	}
}

func TestGo2LeanRepro_ChatToResponsesStreamKeepsRefusal(t *testing.T) {
	chunks := []string{
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":null,"refusal":""}}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"refusal":"I can't help with that."}}]}`,
		`{"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	}
	want := sdkeqChatClient(t, sdkeqSSE(false, chunks...)).Choices[0].Message.Refusal
	c, w := sdkeqGin()
	if _, err := HandleOpenAIChatToResponsesStream(protocol.NewHandleContext(c, "m"), sdkeqChatUpstream(chunks...), "m"); err != nil {
		t.Fatal(err)
	}
	resp := sdkeqResponsesCompleted(t, w.Body.String())
	var refusal string
	for _, item := range resp.Output {
		for _, ct := range item.Content {
			if ct.Type == "refusal" {
				refusal += ct.Refusal
			}
		}
	}
	if refusal != want {
		t.Errorf("responses client refusal %q; upstream refusal (openai-go) %q (output item types %v)", refusal, want, sdkeqTypes(resp))
	}
}

// ---- Finding 70: Responses -> Chat stream, reasoning --------------------------

var sdkeqResponsesReasoningEvents = []string{
	`{"type":"response.created","sequence_number":0,"response":{"id":"resp_1","object":"response","created_at":1,"status":"in_progress","model":"gpt-5","output":[]}}`,
	`{"type":"response.output_item.added","sequence_number":1,"output_index":0,"item":{"id":"rs_1","type":"reasoning","summary":[]}}`,
	`{"type":"response.reasoning_summary_part.added","sequence_number":2,"item_id":"rs_1","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`,
	`{"type":"response.reasoning_summary_text.delta","sequence_number":3,"item_id":"rs_1","output_index":0,"summary_index":0,"delta":"**Greeting** The user says hi."}`,
	`{"type":"response.reasoning_summary_text.done","sequence_number":4,"item_id":"rs_1","output_index":0,"summary_index":0,"text":"**Greeting** The user says hi."}`,
	`{"type":"response.output_item.done","sequence_number":5,"output_index":0,"item":{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"**Greeting** The user says hi."}]}}`,
	`{"type":"response.output_item.added","sequence_number":6,"output_index":1,"item":{"id":"msg_1","type":"message","status":"in_progress","role":"assistant","content":[]}}`,
	`{"type":"response.output_text.delta","sequence_number":7,"item_id":"msg_1","output_index":1,"content_index":0,"delta":"Hello!"}`,
	`{"type":"response.completed","sequence_number":8,"response":{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"gpt-5","output":[{"id":"rs_1","type":"reasoning","summary":[{"type":"summary_text","text":"**Greeting** The user says hi."}]},{"id":"msg_1","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"Hello!","annotations":[]}]}],"usage":{"input_tokens":5,"output_tokens":20,"total_tokens":25,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":12}}}}`,
}

func TestGo2LeanRepro_ResponsesToChatStreamKeepsReasoning(t *testing.T) {
	c, w := sdkeqGin()
	if _, err := HandleResponsesToOpenAIChatStream(protocol.NewHandleContext(c, "m"), sdkeqResponsesUpstream(sdkeqResponsesReasoningEvents...), "m"); err != nil {
		t.Fatal(err)
	}
	// tingly-box's Chat convention for reasoning (as on Anthropic -> Chat) is
	// delta.reasoning_content; openai-go keeps it in the chunk's raw JSON.
	var reasoning string
	for _, data := range sdkeqData(w.Body.String()) {
		var chunk openai.ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			t.Fatal(err)
		}
		for _, ch := range chunk.Choices {
			if f, ok := ch.Delta.JSON.ExtraFields["reasoning_content"]; ok {
				var s string
				_ = json.Unmarshal([]byte(f.Raw()), &s)
				reasoning += s
			}
		}
	}
	if reasoning != "**Greeting** The user says hi." {
		t.Errorf("chat client reasoning_content %q; upstream reasoning summary %q", reasoning, "**Greeting** The user says hi.")
	}
}

// ---- Finding 71: Responses -> Chat stream errors -------------------------------

func TestGo2LeanRepro_ResponsesToChatStreamFailedKeepsUpstreamError(t *testing.T) {
	events := []string{
		`{"type":"response.created","sequence_number":0,"response":{"id":"resp_1","object":"response","created_at":1,"status":"in_progress","model":"gpt-5","output":[]}}`,
		`{"type":"response.in_progress","sequence_number":1,"response":{"id":"resp_1","object":"response","created_at":1,"status":"in_progress","model":"gpt-5","output":[]}}`,
		`{"type":"response.failed","sequence_number":2,"response":{"id":"resp_1","object":"response","created_at":1,"status":"failed","model":"gpt-5","output":[],"error":{"code":"server_error","message":"The model produced invalid content."}}}`,
	}
	c, w := sdkeqGin()
	_, _ = HandleResponsesToOpenAIChatStream(protocol.NewHandleContext(c, "m"), sdkeqResponsesUpstream(events...), "m")
	err := sdkeqChatClientErr(w.Body.String())
	if err == nil || !strings.Contains(err.Error(), "The model produced invalid content.") {
		t.Errorf("chat client (openai-go) stream error %v; want the upstream failure %q", err, "The model produced invalid content.")
	}
}

func TestGo2LeanRepro_ResponsesToChatStreamErrorKeepsCode(t *testing.T) {
	events := []string{
		`{"type":"response.created","sequence_number":0,"response":{"id":"resp_1","object":"response","created_at":1,"status":"in_progress","model":"gpt-5","output":[]}}`,
		`{"type":"error","sequence_number":1,"code":"rate_limit_exceeded","message":"Rate limit reached for gpt-5.","param":null}`,
	}
	c, w := sdkeqGin()
	_, _ = HandleResponsesToOpenAIChatStream(protocol.NewHandleContext(c, "m"), sdkeqResponsesUpstream(events...), "m")
	err := sdkeqChatClientErr(w.Body.String())
	if err == nil || !strings.Contains(err.Error(), "rate_limit_exceeded") {
		t.Errorf("chat client (openai-go) stream error %v; want the upstream code %q", err, "rate_limit_exceeded")
	}
}

// ---- Finding 72: Chat stream tool-call id/name in a later delta -----------------

var sdkeqLateIDChunks = []string{
	`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"role":"assistant"}}]}`,
	`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"type":"function","function":{"arguments":""}}]}}]}`,
	`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]}}]}`,
	`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
}

func TestGo2LeanRepro_ChatStreamLateToolCallIDAndName(t *testing.T) {
	// What openai-go itself makes of the upstream chunks.
	want := sdkeqChatClient(t, sdkeqSSE(false, sdkeqLateIDChunks...)).Choices[0].Message.ToolCalls[0]

	// Responses client.
	c, w := sdkeqGin()
	if _, err := HandleOpenAIChatToResponsesStream(protocol.NewHandleContext(c, "m"), sdkeqChatUpstream(sdkeqLateIDChunks...), "m"); err != nil {
		t.Fatal(err)
	}
	for _, item := range sdkeqResponsesCompleted(t, w.Body.String()).Output {
		if item.Type == "function_call" && (item.Name != want.Function.Name || item.CallID != want.ID) {
			t.Errorf("responses: function_call name %q call_id %q; upstream (openai-go) %q %q", item.Name, item.CallID, want.Function.Name, want.ID)
		}
	}

	// Anthropic client.
	conv := NewOpenAIChatToAnthropicBetaConverter(sdkeqChatUpstream(sdkeqLateIDChunks...), "m", nil)
	var payloads []string
	for {
		ev, done, err := conv.Next()
		if err != nil {
			t.Fatal(err)
		}
		if done {
			break
		}
		if a, ok := AsAnthropicEvent(ev); ok {
			payloads = append(payloads, a.RawJSON())
		}
	}
	for _, b := range sdkeqAnthropicClient(t, payloads).Content {
		if b.Type == "tool_use" && (b.Name != want.Function.Name || b.ID != want.ID) {
			t.Errorf("anthropic: tool_use name %q id %q; upstream (openai-go) %q %q", b.Name, b.ID, want.Function.Name, want.ID)
		}
	}
}

// ---- Finding 73: Gemini thought text shown as the answer, stream ------------------

func TestGo2LeanRepro_GeminiThoughtPartsNotVisibleText(t *testing.T) {
	rs := sdkeqGemini(t,
		`{"candidates":[{"content":{"role":"model","parts":[{"text":"**Planning** The user wants a greeting.","thought":true}]}}]}`,
		`{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello!"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2,"thoughtsTokenCount":9,"totalTokenCount":16}}`)
	// go-genai's own view of the answer: Text() leaves thoughts out.
	var want string
	for _, r := range rs {
		want += r.Text()
	}
	if got := sdkeqGeminiToChat(t, rs).Choices[0].Message.Content; got != want {
		t.Errorf("chat: client content %q; go-genai Text() %q (thought text shown as the answer)", got, want)
	}
	for name, msg := range sdkeqGeminiToAnthropic(t, rs) {
		var text string
		for _, b := range msg.Content {
			if b.Type == "text" {
				text += b.Text
			}
		}
		if text != want {
			t.Errorf("%s: client text %q; go-genai Text() %q (thought text shown as the answer)", name, text, want)
		}
	}
}

// ---- Finding 74: duplicate chat completion ids -------------------------------------

func TestGo2LeanRepro_ChatStreamIDsDistinct(t *testing.T) {
	// Three answers in a row: with ids from a seconds clock, at least two of
	// them share a second.
	rs := sdkeqGemini(t, `{"candidates":[{"content":{"role":"model","parts":[{"text":"Hi"}]},"finishReason":"STOP"}]}`)
	anth := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":1}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":2}}`,
		`{"type":"message_stop"}`,
	}
	paths := map[string]func() string{
		"gemini -> chat": func() string { return sdkeqGeminiToChat(t, rs).ID },
		"anthropic -> chat": func() string {
			c, w := sdkeqGin()
			if _, err := AnthropicToOpenAIStream(protocol.NewHandleContext(c, "m"), nil, sdkeqAnthropicUpstream(anth...), "m", false); err != nil {
				t.Fatal(err)
			}
			return sdkeqChatClient(t, w.Body.String()).ID
		},
	}
	for name, answer := range paths {
		seen := map[string]bool{}
		for range 3 {
			id := answer()
			if seen[id] {
				t.Errorf("%s: two answers share the chat completion id %q; want a distinct id per answer", name, id)
				break
			}
			seen[id] = true
		}
	}
}
