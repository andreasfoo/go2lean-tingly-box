package request

// Reproductions of go2lean findings on the request converters. Each test
// asserts the behaviour a client expects: it fails while the bug is there.

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// findings/issue-1924-tool-call-ids.md
func TestGo2LeanRepro_ToolCallIDsPairAfterConversion(t *testing.T) {
	id := "toolu_" + strings.Repeat("a", 44) // 50 characters, over OpenAI's 40
	req := &anthropic.BetaMessageNewParams{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 1024,
		Messages: []anthropic.BetaMessageParam{
			{Role: anthropic.BetaMessageParamRoleUser, Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock("list the files")}},
			{Role: anthropic.BetaMessageParamRoleAssistant, Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaToolUseBlock(id, map[string]any{"cmd": "ls"}, "bash")}},
			{Role: anthropic.BetaMessageParamRoleUser, Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaToolResultBlock(id, "a.txt", false)}},
		},
	}
	out, _ := ConvertAnthropicBetaToOpenAIRequest(req, false, false, false)
	var callID, resultID string
	for _, m := range out.Messages {
		if m.OfAssistant != nil {
			for _, tc := range m.OfAssistant.ToolCalls {
				if tc.OfFunction != nil {
					callID = tc.OfFunction.ID
				}
			}
		}
		if m.OfTool != nil {
			resultID = m.OfTool.ToolCallID
		}
	}
	if len(callID) > 40 || callID != resultID {
		t.Errorf("tool call id %q (%d chars) and tool message tool_call_id %q (%d chars): want equal and at most 40",
			callID, len(callID), resultID, len(resultID))
	}
}

// findings/issue-1921-system-messages.md (Chat path)
func TestGo2LeanRepro_SystemRoleMessageKeptOnChatPath(t *testing.T) {
	req := &anthropic.BetaMessageNewParams{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 1024,
		Messages: []anthropic.BetaMessageParam{
			{Role: anthropic.BetaMessageParamRole("system"), Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock("Answer in French.")}},
			{Role: anthropic.BetaMessageParamRoleUser, Content: []anthropic.BetaContentBlockParamUnion{anthropic.NewBetaTextBlock("hello")}},
		},
	}
	out, _ := ConvertAnthropicBetaToOpenAIRequest(req, false, false, false)
	for _, m := range out.Messages {
		if m.OfSystem != nil || m.OfDeveloper != nil {
			return
		}
	}
	t.Errorf("the system-role message \"Answer in French.\" is not in the converted request (%d messages)", len(out.Messages))
}

// findings/issue-1921-system-messages.md (Responses path)
func TestGo2LeanRepro_SystemRoleMessageKeptOnResponsesPath(t *testing.T) {
	in := []anthropic.MessageParam{
		{Role: anthropic.MessageParamRole("system"), Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock("Answer in French.")}},
		{Role: anthropic.MessageParamRoleUser, Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock("hello")}},
	}
	out := convertV1MessagesToResponsesInput(in)
	for _, item := range out {
		if item.OfMessage != nil && (item.OfMessage.Role == "system" || item.OfMessage.Role == "developer") {
			return
		}
		if item.OfInputMessage != nil && (item.OfInputMessage.Role == "system" || item.OfInputMessage.Role == "developer") {
			return
		}
	}
	t.Errorf("the system-role message is not among the %d Responses input items", len(out))
}
