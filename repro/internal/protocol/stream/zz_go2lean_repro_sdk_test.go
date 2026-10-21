package stream

// Reproductions checked with the clients' own SDKs as the oracle: what a
// client's SDK makes of tingly-box's converted stream must be what the
// provider said. Each test fails while the bug is there.

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
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
