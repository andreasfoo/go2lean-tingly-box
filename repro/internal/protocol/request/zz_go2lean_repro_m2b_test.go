package request

// Reproduction for finding 61, found by re-searching the go2lean M2 agent's
// own specs with the current model. The test asserts the behaviour a client
// expects: it fails while the bug is there.

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestGo2LeanRepro_GeminiRequestKeepsSampling(t *testing.T) {
	const body = `{"model":"gemini-2.5-pro","max_tokens":256,
	 "temperature":0.2,"top_p":0.9,"top_k":40,"stop_sequences":["END"],
	 "messages":[{"role":"user","content":"Say hi"}]}`
	var v1 anthropic.MessageNewParams
	if err := json.Unmarshal([]byte(body), &v1); err != nil {
		t.Fatal(err)
	}
	var beta anthropic.BetaMessageNewParams
	if err := json.Unmarshal([]byte(body), &beta); err != nil {
		t.Fatal(err)
	}
	_, _, cfgV1 := ConvertAnthropicToGoogleRequest(&v1, 4096)
	_, _, cfgBeta := ConvertAnthropicBetaToGoogleRequest(&beta, 4096)
	for name, cfg := range map[string]struct {
		temp, topP, topK bool
		stop             int
	}{
		"v1":   {cfgV1.Temperature != nil, cfgV1.TopP != nil, cfgV1.TopK != nil, len(cfgV1.StopSequences)},
		"beta": {cfgBeta.Temperature != nil, cfgBeta.TopP != nil, cfgBeta.TopK != nil, len(cfgBeta.StopSequences)},
	} {
		if !cfg.temp || !cfg.topP || !cfg.topK || cfg.stop != 1 {
			t.Errorf("%s: temperature set=%v topP set=%v topK set=%v stopSequences=%d; want all set and 1 stop sequence",
				name, cfg.temp, cfg.topP, cfg.topK, cfg.stop)
		}
	}
}
