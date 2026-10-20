package ops

// Reproductions on the vendor request transforms (Anthropic model and
// metadata transforms, Gemini thinking and schema transforms), found by hand
// (code reading, round 4). Each test asserts the behaviour a client expects:
// it fails while the bug is there.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/openai/openai-go/v3"
)

// findings/anthropic-metadata-empty-account-panic.md: Claude Code without a
// claude.ai login sends metadata.user_id with "account_uuid":"" on
// /v1/messages?beta=true; an API-key provider has no Config.UserID. The Beta
// metadata transform must return a request, as the V1 transform does.
func TestGo2LeanRepro_BetaMetadataEmptyAccountNoPanic(t *testing.T) {
	req := &anthropic.BetaMessageNewParams{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 1024,
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("hi"))},
		Metadata: anthropic.BetaMetadataParam{UserID: param.NewOpt(
			`{"device_id":"6f1d0c2a9b7e4d3c8a5f1e2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6e5f4a3","account_uuid":"","session_id":"0c6b8a3e-1f2d-4e5a-8b7c-9d0e1f2a3b4c"}`)},
	}
	extra := map[string]any{"device": "gateway-device"} // configExtraForMetadata for a provider without an OAuth user id
	var panicked any
	func() {
		defer func() { panicked = recover() }()
		ApplyAnthropicBetaMetadataTransform(req, extra)
	}()
	if panicked != nil {
		t.Errorf("ApplyAnthropicBetaMetadataTransform panicked (%v) on a user_id with an empty account_uuid; want a request back, as ApplyAnthropicV1MetadataTransform returns one", panicked)
	}
}

// findings/anthropic-budget-thinking-stripped.md: for a budget-thinking
// model the model transform strips the signed thinking block from the
// assistant tool-use turn the request continues, and leaves thinking
// enabled; Anthropic rejects that request.
func TestGo2LeanRepro_BetaBudgetThinkingKeepsToolTurnThinking(t *testing.T) {
	req := &anthropic.BetaMessageNewParams{
		Model:     "claude-sonnet-4-5-20250929",
		MaxTokens: 32000,
		Thinking:  anthropic.BetaThinkingConfigParamOfEnabled(10000),
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("list the files")),
			{Role: anthropic.BetaMessageParamRoleAssistant, Content: []anthropic.BetaContentBlockParamUnion{
				anthropic.NewBetaThinkingBlock("EqQBCkYIBxgCKkD...sig", "I should run ls."),
				anthropic.NewBetaToolUseBlock("toolu_01", map[string]any{"command": "ls"}, "Bash"),
			}},
			anthropic.NewBetaUserMessage(anthropic.NewBetaToolResultBlock("toolu_01", "a.go b.go", false)),
		},
	}
	out := ApplyAnthropicBetaModelTransform(req, string(req.Model))
	asst := out.Messages[1].Content
	if out.Thinking.OfEnabled != nil && (len(asst) == 0 || asst[0].OfThinking == nil) {
		b, _ := json.Marshal(out.Messages[1])
		t.Errorf("thinking still enabled (budget %d) but the assistant tool-use turn the request continues is %s: its signed thinking block was stripped; want it kept first (or thinking disabled)", out.Thinking.OfEnabled.BudgetTokens, b)
	}
}

func TestGo2LeanRepro_V1BudgetThinkingKeepsToolTurnThinking(t *testing.T) {
	req := &anthropic.MessageNewParams{
		Model:     "claude-haiku-4-5-20251001",
		MaxTokens: 16000,
		Thinking:  anthropic.ThinkingConfigParamOfEnabled(4096),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("weather?")),
			{Role: anthropic.MessageParamRoleAssistant, Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewThinkingBlock("sig", "call the tool"),
				anthropic.NewToolUseBlock("toolu_1", map[string]any{"city": "Paris"}, "get_weather"),
			}},
			anthropic.NewUserMessage(anthropic.NewToolResultBlock("toolu_1", "sunny", false)),
		},
	}
	out := ApplyAnthropicV1ModelTransform(req, string(req.Model))
	if out.Thinking.OfEnabled != nil && out.Messages[1].Content[0].OfThinking == nil {
		b, _ := json.Marshal(out.Messages[1])
		t.Errorf("thinking still enabled (budget %d) but the assistant tool-use turn the request continues is %s: its signed thinking block was stripped; want it kept first (or thinking disabled)", out.Thinking.OfEnabled.BudgetTokens, b)
	}
}

// findings/gemini-schema-exclusive-bounds.md: exclusiveMinimum is renamed
// to minimum, so with both keys present the result depends on Go map
// iteration order; with the boolean (draft-4 / OpenAPI 3.0) form it can be
// "minimum": true.
func TestGo2LeanRepro_GeminiSchemaExclusiveBoundDeterministic(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		out := filterGeminiSchema(map[string]interface{}{"type": "integer", "minimum": float64(1), "exclusiveMinimum": true})
		b, _ := json.Marshal(out)
		seen[string(b)] = true
	}
	for s := range seen {
		if strings.Contains(s, `"minimum":true`) || len(seen) > 1 {
			var outs []string
			for o := range seen {
				outs = append(outs, o)
			}
			t.Errorf("filterGeminiSchema of {type: integer, minimum: 1, exclusiveMinimum: true} gave %d different outputs %v; want one, with a numeric minimum", len(seen), outs)
			return
		}
	}
}

// findings/gemini-schema-exclusive-bounds.md: subschemas under
// additionalProperties and $defs are not visited, so exclusiveMinimum stays
// there although the filter removes it everywhere it looks.
func TestGo2LeanRepro_GeminiSchemaNestedExclusiveBoundRemoved(t *testing.T) {
	out := filterGeminiSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"weights": map[string]interface{}{"type": "object", "additionalProperties": map[string]interface{}{"type": "number", "exclusiveMinimum": float64(0)}},
			"item":    map[string]interface{}{"$ref": "#/$defs/Item"},
		},
		"$defs": map[string]interface{}{"Item": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"qty": map[string]interface{}{"type": "integer", "exclusiveMinimum": float64(0)}}}},
	})
	b, _ := json.Marshal(out)
	if strings.Contains(string(b), "exclusiveMinimum") {
		t.Errorf("filtered schema %s still has exclusiveMinimum (under additionalProperties and $defs); want it converted at every depth", b)
	}
}

// findings/gemini-thinking-extra-body-overwritten.md: the thinking config
// replaces the whole extra_body, so the client's other extra_body.google
// settings (cached_content) are lost.
func TestGo2LeanRepro_GeminiThinkingKeepsClientExtraBody(t *testing.T) {
	var req openai.ChatCompletionNewParams
	if err := json.Unmarshal([]byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"low"}`), &req); err != nil {
		t.Fatal(err)
	}
	req.SetExtraFields(map[string]any{"extra_body": map[string]any{"google": map[string]any{"cached_content": "cachedContents/abc123"}}})
	out := applyGeminiThinkingConfig(&req, "gemini-2.5-flash", nil)
	eb, _ := out.ExtraFields()["extra_body"].(map[string]interface{})
	g, _ := eb["google"].(map[string]interface{})
	if g["cached_content"] != "cachedContents/abc123" || g["thinking_config"] == nil {
		b, _ := json.Marshal(out.ExtraFields())
		t.Errorf("extra fields after the Gemini thinking transform: %s; want extra_body.google to keep the client's cached_content and gain thinking_config", b)
	}
}
