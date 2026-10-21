package request

// Reproductions checked with the clients' own SDKs as the oracle: the
// upstream request built from a client's history must carry what the
// client's SDK sent. Each test fails while the bug is there.

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	"github.com/tingly-dev/tingly-box/internal/protocol"
)

// sdkeqChatReq is a Chat client's history replaying one tool call with the
// given arguments string.
func sdkeqChatReq(t *testing.T, args string) *openai.ChatCompletionNewParams {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	body := `{"model":"claude-sonnet-4-5","messages":[` +
		`{"role":"user","content":"Look it up"},` +
		`{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_message","arguments":` + string(argsJSON) + `}}]},` +
		`{"role":"tool","tool_call_id":"call_1","content":"ok"}]}`
	var req openai.ChatCompletionNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

// sdkeqResponsesReq is the same history from a Responses client, parsed as
// the gateway parses a Responses request body.
func sdkeqResponsesReq(t *testing.T, args string) *responses.ResponseNewParams {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	body := `{"model":"claude-sonnet-4-5","input":[` +
		`{"role":"user","content":"Look it up"},` +
		`{"type":"function_call","call_id":"call_1","name":"get_message","arguments":` + string(argsJSON) + `},` +
		`{"type":"function_call_output","call_id":"call_1","output":"ok"}]}`
	var req protocol.ResponseCreateRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return req.ResponseNewParams
}

// sdkeqToolUseInputs marshals the converted Anthropic request (what goes on
// the wire) and returns each tool_use block's raw input ("" when absent).
func sdkeqToolUseInputs(t *testing.T, req any) []string {
	t.Helper()
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Messages []struct {
			Content []map[string]json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, m := range wire.Messages {
		for _, b := range m.Content {
			if string(b["type"]) == `"tool_use"` {
				out = append(out, string(b["input"]))
			}
		}
	}
	return out
}

func sdkeqIDNumber(t *testing.T, raw string) string {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader([]byte(raw)))
	d.UseNumber()
	var m map[string]any
	if err := d.Decode(&m); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	n, _ := m["id"].(json.Number)
	return n.String()
}

// ---- Finding 65: tool_use without input --------------------------------------

func TestGo2LeanRepro_ToolUseInputPresentInAnthropicRequest(t *testing.T) {
	// "" is what a parameterless call streams as (finding 24, and other
	// gateways/providers); the others are malformed or non-object arguments.
	for _, args := range []string{"", `{"city":`, "null"} {
		for name, req := range map[string]any{
			"chat":      ConvertOpenAIToAnthropicRequest(sdkeqChatReq(t, args), 4096),
			"responses": ConvertOpenAIResponsesToAnthropicBetaRequest(*sdkeqResponsesReq(t, args), 4096),
		} {
			for _, in := range sdkeqToolUseInputs(t, req) {
				var v any
				_ = json.Unmarshal([]byte(in), &v)
				if _, ok := v.(map[string]any); !ok {
					if in == "" {
						in = "(no input key)"
					}
					t.Errorf("%s, arguments %q: tool_use input %s; Anthropic requires an object (request rejected: input: Field required)", name, args, in)
				}
			}
		}
	}
}

// ---- Finding 66: tool-argument numbers rounded, request ---------------------------

func TestGo2LeanRepro_ToolArgsNumbersExactInAnthropicRequest(t *testing.T) {
	const args = `{"id":9007199254740993}`
	for name, req := range map[string]any{
		"chat":      ConvertOpenAIToAnthropicRequest(sdkeqChatReq(t, args), 4096),
		"responses": ConvertOpenAIResponsesToAnthropicBetaRequest(*sdkeqResponsesReq(t, args), 4096),
	} {
		inputs := sdkeqToolUseInputs(t, req)
		if len(inputs) != 1 {
			t.Fatalf("%s: tool_use blocks %v", name, inputs)
		}
		if got := sdkeqIDNumber(t, inputs[0]); got != "9007199254740993" {
			t.Errorf("%s: tool_use input %s; the client sent arguments %s", name, inputs[0], args)
		}
	}
}
