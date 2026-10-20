package ops

import (
	"github.com/anthropics/anthropic-sdk-go"
)

// Hand-written go2lean specs for the vendor request transforms (findings
// 35-40, round 4). Finding 35 needs no spec: go2lean's no-panic property
// of ApplyAnthropicBetaMetadataTransform states it.

// go2lean:spec ApplyAnthropicBetaModelTransform
// SpecApplyAnthropicBetaModelTransformToolTurnThinking: a request that
// leaves thinking enabled and continues a tool turn (it ends with tool
// results) keeps the Anthropic API's rule: the last assistant message
// starts with a thinking or redacted_thinking block.
func SpecApplyAnthropicBetaModelTransformToolTurnThinking(req *anthropic.BetaMessageNewParams, model string, ret *anthropic.BetaMessageNewParams) bool {
	if ret == nil || ret.Thinking.OfEnabled == nil || len(ret.Messages) < 2 {
		return true
	}
	last := ret.Messages[len(ret.Messages)-1]
	if string(last.Role) != "user" {
		return true
	}
	results := false
	for _, b := range last.Content {
		if b.OfToolResult != nil {
			results = true
		}
	}
	if !results {
		return true
	}
	for i := len(ret.Messages) - 2; i >= 0; i-- {
		m := ret.Messages[i]
		if string(m.Role) != "assistant" {
			continue
		}
		if len(m.Content) == 0 {
			return false
		}
		return m.Content[0].OfThinking != nil || m.Content[0].OfRedactedThinking != nil
	}
	return true
}

// hasExclusiveBound says whether a JSON schema holds exclusiveMinimum or
// exclusiveMaximum at any depth.
func hasExclusiveBound(v any, depth int) bool {
	if depth > 6 {
		return false
	}
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			if k == "exclusiveMinimum" || k == "exclusiveMaximum" || hasExclusiveBound(e, depth+1) {
				return true
			}
		}
	case []any:
		for _, e := range x {
			if hasExclusiveBound(e, depth+1) {
				return true
			}
		}
	}
	return false
}

// go2lean:spec filterGeminiSchema
// SpecFilterGeminiSchemaNoExclusiveBounds: the schema sent to Gemini holds
// no exclusiveMinimum or exclusiveMaximum at any depth (Gemini rejects
// them; the filter exists to remove them).
func SpecFilterGeminiSchemaNoExclusiveBounds(schema map[string]interface{}, ret map[string]interface{}) bool {
	return !hasExclusiveBound(ret, 0)
}
