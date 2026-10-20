package request

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"

	"github.com/tingly-dev/tingly-box/internal/protocol"
)

// Variant of SpecConvertAnthropicBetaToOpenAIRequestKeepsToolSchema (fix
// verification, not part of go2lean-tingly-box): tools are matched by
// position among the tools with the same name, so two tools sharing a name
// (which the Anthropic API rejects) are no longer compared with each other.

// go2lean:spec ConvertAnthropicBetaToOpenAIRequest
// SpecFixwsAnthropicBetaToOpenAIRequestKeepsToolSchemaUniqueNames: with
// unique tool names, every input_schema keyword beyond type/properties/
// required reaches the Chat function's parameters.
func SpecFixwsAnthropicBetaToOpenAIRequestKeepsToolSchemaUniqueNames(anthropicReq *anthropic.BetaMessageNewParams, compatible bool, isStreaming bool, disableStreamUsage bool, ret0 *openai.ChatCompletionNewParams, ret1 *protocol.OpenAIConfig) bool {
	if anthropicReq == nil || ret0 == nil {
		return true
	}
	seen := map[string]bool{}
	for _, t := range anthropicReq.Tools {
		if t.OfTool == nil {
			continue
		}
		if seen[t.OfTool.Name] {
			return true // duplicate tool names: out of scope
		}
		seen[t.OfTool.Name] = true
	}
	for _, t := range anthropicReq.Tools {
		if t.OfTool == nil {
			continue
		}
		for _, out := range ret0.Tools {
			if out.OfFunction == nil || out.OfFunction.Function.Name != t.OfTool.Name {
				continue
			}
			for k := range t.OfTool.InputSchema.ExtraFields {
				if _, ok := out.OfFunction.Function.Parameters[k]; !ok {
					return false
				}
			}
		}
	}
	return true
}
