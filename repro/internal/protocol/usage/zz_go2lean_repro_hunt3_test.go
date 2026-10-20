package usage

// Reproductions on the usage accumulator, found by hand (code reading,
// round 3). Each test asserts the behaviour a client expects: it fails while
// the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func reproH3BetaEvent(t *testing.T, body string) *anthropic.BetaRawMessageStreamEventUnion {
	t.Helper()
	var ev anthropic.BetaRawMessageStreamEventUnion
	if err := json.Unmarshal([]byte(body), &ev); err != nil {
		t.Fatal(err)
	}
	return &ev
}

// findings/usage-anthropic-delta-cache-creation.md: message_start reports
// 10 uncached input tokens and 100 cache-creation tokens; message_delta
// usage is cumulative and may repeat any subset of the input fields.
const reproH3Start = `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude",
 "usage":{"input_tokens":10,"cache_creation_input_tokens":100,"cache_read_input_tokens":0,"output_tokens":1}}}`

func TestGo2LeanRepro_AccumulatorDeltaRepeatingCacheCreation(t *testing.T) {
	acc := NewAnthropicAccumulator()
	acc.ConsumeBeta(reproH3BetaEvent(t, reproH3Start))
	acc.ConsumeBeta(reproH3BetaEvent(t, `{"type":"message_delta","delta":{"stop_reason":"end_turn"},
	 "usage":{"output_tokens":5,"cache_creation_input_tokens":100}}`))
	if u := acc.Result(); u.InputTokens != 110 {
		t.Errorf("InputTokens = %d, want 110 (input 10 + cache creation 100, the creation counted once): %+v", u.InputTokens, *u)
	}
}

func TestGo2LeanRepro_AccumulatorDeltaRepeatingInput(t *testing.T) {
	acc := NewAnthropicAccumulator()
	acc.ConsumeBeta(reproH3BetaEvent(t, reproH3Start))
	acc.ConsumeBeta(reproH3BetaEvent(t, `{"type":"message_delta","delta":{"stop_reason":"end_turn"},
	 "usage":{"input_tokens":10,"output_tokens":5}}`))
	if u := acc.Result(); u.InputTokens != 110 || u.CacheWriteTokens > u.InputTokens {
		t.Errorf("InputTokens = %d, CacheWriteTokens = %d, want 110 with the writes a subset of the input: %+v", u.InputTokens, u.CacheWriteTokens, *u)
	}
}
