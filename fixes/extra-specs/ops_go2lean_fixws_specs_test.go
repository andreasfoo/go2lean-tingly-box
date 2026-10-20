package ops

import (
	"github.com/anthropics/anthropic-sdk-go"
)

// Variant of SpecApplyAnthropicBetaModelTransformToolTurnThinking (fix
// verification, not part of go2lean-tingly-box): the same API rule, checked
// only for a well-formed tool turn, i.e. the last assistant message holds a
// tool_use (the tool_result must answer one).

// go2lean:spec ApplyAnthropicBetaModelTransform
// SpecFixwsBetaModelTransformToolUseTurnThinking: a request that leaves
// thinking enabled and continues a tool-use turn (assistant tool_use, then
// user tool_result) has the assistant message start with a thinking or
// redacted_thinking block.
func SpecFixwsBetaModelTransformToolUseTurnThinking(req *anthropic.BetaMessageNewParams, model string, ret *anthropic.BetaMessageNewParams) bool {
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
		toolUse := false
		for _, b := range m.Content {
			if b.OfToolUse != nil {
				toolUse = true
			}
		}
		if !toolUse {
			return true // malformed: a tool_result answering no tool_use
		}
		return m.Content[0].OfThinking != nil || m.Content[0].OfRedactedThinking != nil
	}
	return true
}
