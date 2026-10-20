package stream

// Reproduction on the Gemini -> Anthropic v1 stream for finding 59
// (gemini-anthropic-safety-content-filter), found by the go2lean M2 agent
// with its own spec on the non-stream mapper (a Claude agent, Opus 5.5, that
// had never seen the findings wrote it from the package source and go2lean's
// agent prompt); the stream uses the same mapper, found by reading and
// confirmed here. The test asserts the behaviour a client expects: it fails
// while the bug is there.

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

func TestGo2LeanRepro_StreamGeminiSafetyIsRefusalForAnthropicV1(t *testing.T) {
	// Gemini streams part of the answer, then blocks it: the last chunk has
	// no content, only the finish reason and the rating that blocked it.
	var text, blocked genai.GenerateContentResponse
	if err := json.Unmarshal([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"Here is how"}]},"index":0}],
	 "usageMetadata":{"promptTokenCount":12,"totalTokenCount":12},"modelVersion":"gemini-2.5-flash"}`), &text); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"candidates":[{"finishReason":"SAFETY","index":0,
	 "safetyRatings":[{"category":"HARM_CATEGORY_DANGEROUS_CONTENT","probability":"HIGH","blocked":true}]}],
	 "usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":3,"totalTokenCount":15},"modelVersion":"gemini-2.5-flash"}`), &blocked); err != nil {
		t.Fatal(err)
	}
	if len(blocked.Candidates) != 1 || blocked.Candidates[0].FinishReason != genai.FinishReasonSafety {
		t.Fatalf("fixture not parsed: %+v", blocked.Candidates)
	}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	chunks := func(yield func(*genai.GenerateContentResponse, error) bool) {
		_ = yield(&text, nil) && yield(&blocked, nil)
	}
	if _, err := HandleGoogleToAnthropicStreamResponse(c, chunks, "m"); err != nil {
		t.Fatal(err)
	}
	stop := ""
	for _, line := range strings.Split(w.Body.String(), "\n") {
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		var ev struct {
			Type  string `json:"type"`
			Delta struct {
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(data)), &ev) == nil && ev.Type == "message_delta" {
			stop = ev.Delta.StopReason
		}
	}
	if stop != "refusal" {
		t.Errorf("Anthropic v1 stream: Gemini SAFETY ends with stop_reason %q, not an Anthropic stop reason; clients that check for \"refusal\" miss the block", stop)
	}
}
