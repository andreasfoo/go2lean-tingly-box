package request

// Reproductions of the bugs found by go2lean from the issue-derived
// specs (run "issues1"): each spec carries the class of a tingly-box issue
// to another conversion direction. Each test fails while the bug is there.

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"google.golang.org/genai"

	"github.com/tingly-dev/tingly-box/internal/protocol"
)

func issues1Unmarshal(t *testing.T, body string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("parse %s: %v", body, err)
	}
}

// issues1GoogleText is all the text of a Gemini request: system
// instruction and contents.
func issues1GoogleText(contents []*genai.Content, config *genai.GenerateContentConfig) string {
	var b strings.Builder
	if config != nil && config.SystemInstruction != nil {
		for _, p := range config.SystemInstruction.Parts {
			b.WriteString(p.Text)
		}
	}
	for _, c := range contents {
		for _, p := range c.Parts {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// ---- Finding 75: inline system messages dropped on Anthropic -> Gemini (#1921 class) ----

// Claude Code 2.1 sends its environment block as a role "system" message
// inside messages[] (issue #1921); on a Google-style provider it is lost.
const issues1ClaudeCodeInlineSystem = `{"model":"gemini-2.5-pro","max_tokens":1024,"messages":[` +
	`{"role":"user","content":"hi"},` +
	`{"role":"system","content":[{"type":"text","text":"# Environment\n - Primary working directory: /work/proj\n - Platform: linux"}]}]}`

func TestGo2LeanRepro_AnthropicToGeminiKeepsInlineSystemMessages(t *testing.T) {
	var v1 anthropic.MessageNewParams
	issues1Unmarshal(t, issues1ClaudeCodeInlineSystem, &v1)
	var beta anthropic.BetaMessageNewParams
	issues1Unmarshal(t, issues1ClaudeCodeInlineSystem, &beta)
	_, c1, cfg1 := ConvertAnthropicToGoogleRequest(&v1, 1024)
	_, c2, cfg2 := ConvertAnthropicBetaToGoogleRequest(&beta, 1024)
	for name, text := range map[string]string{"v1": issues1GoogleText(c1, cfg1), "beta": issues1GoogleText(c2, cfg2)} {
		if !strings.Contains(text, "Primary working directory: /work/proj") {
			t.Errorf("%s: the inline system message is missing from the Gemini request (text %q)", name, text)
		}
	}
}

// ---- Finding 76: foreign tool-call ids sent to Anthropic unchanged (#1401 class) ----

var issues1AnthropicID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// A history produced by Kimi (tool call ids "functions.<name>:<n>"), then
// routed to an Anthropic provider; api.anthropic.com rejects tool_use.id /
// tool_result.tool_use_id outside ^[a-zA-Z0-9_-]+$.
func TestGo2LeanRepro_ForeignToolIDsSanitizedForAnthropic(t *testing.T) {
	var chat openai.ChatCompletionNewParams
	issues1Unmarshal(t, `{"model":"claude-sonnet-4-5","messages":[`+
		`{"role":"user","content":"Read main.go"},`+
		`{"role":"assistant","content":"","tool_calls":[{"id":"functions.read_file:0","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]},`+
		`{"role":"tool","tool_call_id":"functions.read_file:0","content":"package main"}]}`, &chat)
	var rr protocol.ResponseCreateRequest
	issues1Unmarshal(t, `{"model":"claude-sonnet-4-5","input":[`+
		`{"role":"user","content":"Read main.go"},`+
		`{"type":"function_call","call_id":"functions.read_file:0","name":"read_file","arguments":"{\"path\":\"main.go\"}"},`+
		`{"type":"function_call_output","call_id":"functions.read_file:0","output":"package main"}]}`, &rr)

	for name, req := range map[string]*anthropic.BetaMessageNewParams{
		"chat":      ConvertOpenAIToAnthropicRequest(&chat, 4096),
		"responses": ConvertOpenAIResponsesToAnthropicBetaRequest(*rr.ResponseNewParams, 4096),
	} {
		var useID, resultID string
		for _, m := range req.Messages {
			for _, b := range m.Content {
				if b.OfToolUse != nil {
					useID = b.OfToolUse.ID
				}
				if b.OfToolResult != nil {
					resultID = b.OfToolResult.ToolUseID
				}
			}
		}
		if !issues1AnthropicID.MatchString(useID) || !issues1AnthropicID.MatchString(resultID) {
			t.Errorf("%s: tool_use.id %q, tool_result.tool_use_id %q; Anthropic requires ^[a-zA-Z0-9_-]+$ (request rejected with 400)", name, useID, resultID)
		}
		if useID != resultID {
			t.Errorf("%s: tool_use.id %q and tool_result.tool_use_id %q no longer pair", name, useID, resultID)
		}
	}
}

// ---- Finding 77: Responses -> Anthropic sends an empty text block (#1251 class) ----

// A Responses user turn whose only part is a PDF (input_file) or an image
// by file_id: every part is skipped and the converter sends a user message
// with one empty text block, which Anthropic rejects ("text content blocks
// must be non-empty"), failing the whole request.
func TestGo2LeanRepro_ResponsesToAnthropicNoEmptyTextBlock(t *testing.T) {
	for name, part := range map[string]string{
		"input_file": `{"type":"input_file","filename":"report.pdf","file_data":"data:application/pdf;base64,JVBERi0xLjQK"}`,
		"file_id":    `{"type":"input_image","file_id":"file-abc123","detail":"auto"}`,
	} {
		var rr protocol.ResponseCreateRequest
		issues1Unmarshal(t, `{"model":"claude-sonnet-4-5","input":[{"role":"user","content":[`+part+`]}]}`, &rr)
		req := ConvertOpenAIResponsesToAnthropicBetaRequest(*rr.ResponseNewParams, 4096)
		for i, m := range req.Messages {
			for _, b := range m.Content {
				if b.OfText != nil && b.OfText.Text == "" {
					raw, _ := json.Marshal(m)
					t.Errorf("%s: messages[%d] = %s; Anthropic rejects an empty text block", name, i, raw)
				}
			}
		}
	}
}

// ---- Finding 82: Gemini -> Anthropic empty message, no gateway caller (#1251 class) ----

func TestGo2LeanRepro_GeminiToAnthropicNoEmptyMessage(t *testing.T) {
	contents := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("\x89PNG\r\n")}}}},
		{Role: "user", Parts: []*genai.Part{{Text: "What is in this picture?"}}},
	}
	req := ConvertGoogleToAnthropicRequest("claude-sonnet-4-5", contents, nil)
	for i, m := range req.Messages {
		if len(m.Content) == 0 {
			raw, _ := json.Marshal(m)
			t.Errorf("messages[%d] = %s; Anthropic rejects a message with empty content", i, raw)
		}
	}
}

// ---- Finding 78: tool_result images dropped on Anthropic -> Gemini (#1613 follow-up) ----

func TestGo2LeanRepro_AnthropicToGeminiToolResultImage(t *testing.T) {
	body := `{"model":"gemini-2.5-pro","max_tokens":1024,"messages":[` +
		`{"role":"user","content":"Take a screenshot"},` +
		`{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"screenshot","input":{}}]},` +
		`{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":[` +
		`{"type":"text","text":"screenshot taken"},` +
		`{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}]}]}]}`
	var v1 anthropic.MessageNewParams
	issues1Unmarshal(t, body, &v1)
	var beta anthropic.BetaMessageNewParams
	issues1Unmarshal(t, body, &beta)
	_, c1, _ := ConvertAnthropicToGoogleRequest(&v1, 1024)
	_, c2, _ := ConvertAnthropicBetaToGoogleRequest(&beta, 1024)
	for name, contents := range map[string][]*genai.Content{"v1": c1, "beta": c2} {
		found := false
		for _, c := range contents {
			for _, p := range c.Parts {
				if p.InlineData != nil {
					found = true
				}
				if p.FunctionResponse != nil {
					for _, fp := range p.FunctionResponse.Parts {
						if fp.InlineData != nil {
							found = true
						}
					}
				}
			}
		}
		if !found {
			raw, _ := json.Marshal(contents)
			t.Errorf("%s: the tool_result image is not in the Gemini request: %s", name, raw)
		}
	}
}

// ---- Finding 79: thinking intent not carried to Gemini (#1897 / #1917 class) ----

func TestGo2LeanRepro_ThinkingIntentReachesGemini(t *testing.T) {
	anth := `{"model":"gemini-2.5-flash","max_tokens":16000,"thinking":{"type":"enabled","budget_tokens":10240},"messages":[{"role":"user","content":"Prove that sqrt(2) is irrational."}]}`
	var v1 anthropic.MessageNewParams
	issues1Unmarshal(t, anth, &v1)
	var beta anthropic.BetaMessageNewParams
	issues1Unmarshal(t, anth, &beta)
	var chat openai.ChatCompletionNewParams
	issues1Unmarshal(t, `{"model":"gemini-2.5-flash","reasoning_effort":"high","messages":[{"role":"user","content":"Prove that sqrt(2) is irrational."}]}`, &chat)
	_, _, cfg1 := ConvertAnthropicToGoogleRequest(&v1, 4096)
	_, _, cfg2 := ConvertAnthropicBetaToGoogleRequest(&beta, 4096)
	_, _, cfg3 := ConvertOpenAIToGoogleRequest(&chat, 4096)
	for name, cfg := range map[string]*genai.GenerateContentConfig{"anthropic v1": cfg1, "anthropic beta": cfg2, "chat": cfg3} {
		if cfg.ThinkingConfig == nil {
			t.Errorf("%s: the client asked for thinking, the Gemini request has no thinkingConfig", name)
		}
	}
}

// ---- Finding 47, further trigger: assistant array-form text dropped (#1427 class) ----

func TestGo2LeanRepro_ChatToGeminiAssistantArrayText(t *testing.T) {
	var chat openai.ChatCompletionNewParams
	issues1Unmarshal(t, `{"model":"gemini-2.5-pro","messages":[`+
		`{"role":"user","content":"Pick a number"},`+
		`{"role":"assistant","content":[{"type":"text","text":"I pick 42."}]},`+
		`{"role":"user","content":"Which number did you pick?"}]}`, &chat)
	_, contents, cfg := ConvertOpenAIToGoogleRequest(&chat, 4096)
	if text := issues1GoogleText(contents, cfg); !strings.Contains(text, "I pick 42.") {
		raw, _ := json.Marshal(contents)
		t.Errorf("the assistant turn is missing from the Gemini request: %s", raw)
	}
}
