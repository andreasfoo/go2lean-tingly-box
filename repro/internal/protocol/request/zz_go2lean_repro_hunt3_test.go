package request

// Reproductions on the request conversions, found by hand (code reading,
// round 3), except the two tool_choice "none" tests at the end (findings
// 33-34), whose bugs the go2lean M2 agent found with its own spec
// (MiniMax-M3); the tests were written for them afterwards. Each test
// asserts the behaviour a client expects: it fails while the bug is there.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

func reproH3JSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func reproH3Anthropic(t *testing.T, body string) *anthropic.BetaMessageNewParams {
	t.Helper()
	var req anthropic.BetaMessageNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func reproH3Chat(t *testing.T, body string) *openai.ChatCompletionNewParams {
	t.Helper()
	var req openai.ChatCompletionNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func reproH3Responses(t *testing.T, body string) *responses.ResponseNewParams {
	t.Helper()
	var req responses.ResponseNewParams
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

// findings/anthropic-tool-schema-keywords.md: a pydantic/FastMCP-style tool
// schema (nested model in $defs, referenced by $ref, closed with
// additionalProperties:false), as Claude Code forwards MCP tools.
const reproH3SchemaBody = `{"model":"m","max_tokens":64,
"tools":[{"name":"create_event","description":"d","input_schema":{
  "type":"object",
  "$defs":{"Attendee":{"type":"object","properties":{"email":{"type":"string"}},"required":["email"]}},
  "properties":{"attendees":{"type":"array","items":{"$ref":"#/$defs/Attendee"}}},
  "required":["attendees"],
  "additionalProperties":false}}],
"messages":[{"role":"user","content":"hi"}]}`

func TestGo2LeanRepro_AnthropicToChatToolSchemaKeepsKeywords(t *testing.T) {
	out, _ := ConvertAnthropicBetaToOpenAIRequest(reproH3Anthropic(t, reproH3SchemaBody), false, false, false)
	params := gjson.Get(reproH3JSON(t, out), "tools.0.function.parameters")
	if !params.Get(`\$defs`).Exists() || !params.Get("additionalProperties").Exists() {
		t.Errorf("Chat tool parameters lost $defs/additionalProperties (the $ref now dangles): %s", params.Raw)
	}
}

func TestGo2LeanRepro_AnthropicToResponsesToolSchemaKeepsKeywords(t *testing.T) {
	params := gjson.Get(reproH3JSON(t, ConvertAnthropicBetaToResponsesRequest(reproH3Anthropic(t, reproH3SchemaBody))), "tools.0.parameters")
	if !params.Get(`\$defs`).Exists() || !params.Get("additionalProperties").Exists() {
		t.Errorf("beta -> Responses tool parameters lost $defs/additionalProperties: %s", params.Raw)
	}
	var v1 anthropic.MessageNewParams
	if err := json.Unmarshal([]byte(reproH3SchemaBody), &v1); err != nil {
		t.Fatal(err)
	}
	params1 := gjson.Get(reproH3JSON(t, ConvertAnthropicV1ToResponsesRequest(&v1)), "tools.0.parameters")
	if !params1.Get(`\$defs`).Exists() || !params1.Get("additionalProperties").Exists() {
		t.Errorf("v1 -> Responses tool parameters lost $defs/additionalProperties: %s", params1.Raw)
	}
}

// findings/structured-output-dropped.md: a JSON-schema output format, in
// each client protocol's own field.
const reproH3OutSchema = `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`

func TestGo2LeanRepro_ChatResponseFormatCarried(t *testing.T) {
	req := reproH3Chat(t, `{"model":"m","messages":[{"role":"user","content":"give json"}],
	"response_format":{"type":"json_schema","json_schema":{"name":"answer","strict":true,"schema":`+reproH3OutSchema+`}}}`)
	if w := reproH3JSON(t, ConvertChatToOpenAIResponses(req, 0)); !gjson.Get(w, "text.format").Exists() {
		t.Errorf("Chat -> Responses: response_format dropped, want text.format: %s", w)
	}
	if w := reproH3JSON(t, ConvertOpenAIToAnthropicRequest(req, 1024)); !gjson.Get(w, "output_config.format").Exists() {
		t.Errorf("Chat -> Anthropic: response_format dropped, want output_config.format: %s", w)
	}
}

func TestGo2LeanRepro_ResponsesTextFormatCarried(t *testing.T) {
	req := reproH3Responses(t, `{"model":"m","input":"give json",
	"text":{"format":{"type":"json_schema","name":"answer","strict":true,"schema":`+reproH3OutSchema+`}}}`)
	if w := reproH3JSON(t, ConvertOpenAIResponsesToChat(req, 0)); !gjson.Get(w, "response_format").Exists() {
		t.Errorf("Responses -> Chat: text.format dropped, want response_format: %s", w)
	}
	if w := reproH3JSON(t, ConvertOpenAIResponsesToAnthropicBetaRequest(*req, 1024)); !gjson.Get(w, "output_config.format").Exists() {
		t.Errorf("Responses -> Anthropic: text.format dropped, want output_config.format: %s", w)
	}
}

func TestGo2LeanRepro_AnthropicOutputFormatCarried(t *testing.T) {
	req := reproH3Anthropic(t, `{"model":"m","max_tokens":256,"messages":[{"role":"user","content":"give json"}],
	"output_config":{"format":{"type":"json_schema","schema":`+reproH3OutSchema+`}}}`)
	chat, _ := ConvertAnthropicBetaToOpenAIRequest(req, false, false, false)
	if w := reproH3JSON(t, chat); !gjson.Get(w, "response_format").Exists() {
		t.Errorf("Anthropic -> Chat: output_config.format dropped, want response_format: %s", w)
	}
	if w := reproH3JSON(t, ConvertAnthropicBetaToResponsesRequest(req)); !gjson.Get(w, "text.format").Exists() {
		t.Errorf("Anthropic -> Responses: output_config.format dropped, want text.format: %s", w)
	}
}

// findings/document-content-dropped.md: a PDF sent as an Anthropic document
// block or a Chat file part.
const reproH3PDF = "JVBERi0xLjQK" // "%PDF-1.4\n"

func TestGo2LeanRepro_AnthropicDocumentBlockCarried(t *testing.T) {
	req := reproH3Anthropic(t, `{"model":"m","max_tokens":64,"messages":[{"role":"user","content":[
	 {"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"`+reproH3PDF+`"}},
	 {"type":"text","text":"Summarize this PDF."}]}]}`)
	chat, _ := ConvertAnthropicBetaToOpenAIRequest(req, false, false, false)
	if w := reproH3JSON(t, chat); !strings.Contains(w, reproH3PDF) {
		t.Errorf("Anthropic -> Chat: document block dropped (want a file part): %s", gjson.Get(w, "messages").Raw)
	}
	if w := reproH3JSON(t, ConvertAnthropicBetaToResponsesRequest(req)); !strings.Contains(w, reproH3PDF) {
		t.Errorf("Anthropic -> Responses: document block dropped (want an input_file part): %s", gjson.Get(w, "input").Raw)
	}
}

func TestGo2LeanRepro_ChatFilePartCarried(t *testing.T) {
	req := reproH3Chat(t, `{"model":"m","messages":[{"role":"user","content":[
	 {"type":"file","file":{"filename":"a.pdf","file_data":"data:application/pdf;base64,`+reproH3PDF+`"}},
	 {"type":"text","text":"Summarize this PDF."}]}]}`)
	if w := reproH3JSON(t, ConvertOpenAIToAnthropicRequest(req, 1024)); !strings.Contains(w, reproH3PDF) {
		t.Errorf("Chat -> Anthropic: file part dropped (want a document block): %s", gjson.Get(w, "messages").Raw)
	}
	if w := reproH3JSON(t, ConvertChatToOpenAIResponses(req, 0)); !strings.Contains(w, reproH3PDF) {
		t.Errorf("Chat -> Responses: file part dropped (want an input_file part): %s", gjson.Get(w, "input").Raw)
	}
}

// findings/responses-chat-split-assistant-turn.md: one assistant turn of a
// Codex history (preamble text, then a function_call). The assistant item
// is in the string form the gateway's PreprocessInputData flattens Codex's
// output_text history into.
func TestGo2LeanRepro_ResponsesAssistantTurnIsOneChatMessage(t *testing.T) {
	req := reproH3Responses(t, `{"model":"m","input":[
	 {"type":"message","role":"user","content":"What's in /tmp?"},
	 {"type":"message","role":"assistant","content":"I'll list the directory."},
	 {"type":"function_call","call_id":"call_1","name":"shell","arguments":"{\"cmd\":\"ls /tmp\"}"},
	 {"type":"function_call_output","call_id":"call_1","output":"a.txt"}]}`)
	w := reproH3JSON(t, ConvertOpenAIResponsesToChat(req, 0))
	msgs := gjson.Get(w, "messages").Array()
	for i := 1; i < len(msgs); i++ {
		if msgs[i].Get("role").String() == "assistant" && msgs[i-1].Get("role").String() == "assistant" {
			t.Errorf("messages %d and %d are both assistant (want one assistant message with content and tool_calls): %s", i-1, i, gjson.Get(w, "messages").Raw)
		}
	}
}

// findings/responses-anthropic-tool-output-image.md: a screenshot tool
// returns text and an image.
func TestGo2LeanRepro_ToolOutputImageReachesAnthropic(t *testing.T) {
	req := reproH3Responses(t, `{"model":"m","input":[
	 {"type":"function_call","call_id":"call_1","name":"screenshot","arguments":"{}"},
	 {"type":"function_call_output","call_id":"call_1","output":[
	   {"type":"input_text","text":"here"},
	   {"type":"input_image","image_url":"data:image/png;base64,iVBORw0KGgo="}]}]}`)
	w := reproH3JSON(t, ConvertOpenAIResponsesToAnthropicBetaRequest(*req, 1024))
	if !strings.Contains(w, "iVBORw0KGgo=") {
		t.Errorf("tool output image dropped (want an image block in tool_result.content): %s", gjson.Get(w, "messages").Raw)
	}
}

// findings/chat-responses-tool-strict.md: the strict flag of a function
// tool, in both directions.
func TestGo2LeanRepro_ChatToResponsesKeepsStrict(t *testing.T) {
	req := reproH3Chat(t, `{"model":"m","messages":[{"role":"user","content":"hi"}],
	"tools":[{"type":"function","function":{"name":"search","strict":false,"parameters":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer"}},"required":["q"]}}}]}`)
	w := reproH3JSON(t, ConvertChatToOpenAIResponses(req, 0))
	if s := gjson.Get(w, "tools.0.strict"); !s.Exists() || s.Bool() {
		t.Errorf("Chat -> Responses: strict:false not carried: %s", gjson.Get(w, "tools").Raw)
	}
}

func TestGo2LeanRepro_ResponsesToChatKeepsStrict(t *testing.T) {
	req := reproH3Responses(t, `{"model":"m","input":"hi",
	"tools":[{"type":"function","name":"search","strict":true,"parameters":{"type":"object","properties":{"q":{"type":"string"}},"required":["q"],"additionalProperties":false}}]}`)
	w := reproH3JSON(t, ConvertOpenAIResponsesToChat(req, 0))
	if !gjson.Get(w, "tools.0.function.strict").Bool() {
		t.Errorf("Responses -> Chat: strict:true not carried: %s", gjson.Get(w, "tools").Raw)
	}
}

// findings/responses-anthropic-tool-choice-none.md (found by the go2lean
// M2 agent): a Responses request that defines tools and forbids calling
// them.
func TestGo2LeanRepro_ResponsesToolChoiceNoneReachesAnthropic(t *testing.T) {
	req := reproH3Responses(t, `{"model":"m","input":"Summarize the conversation so far.",
	"tools":[{"type":"function","name":"shell","strict":false,"parameters":{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}}],
	"tool_choice":"none"}`)
	w := reproH3JSON(t, ConvertOpenAIResponsesToAnthropicBetaRequest(*req, 1024))
	if !gjson.Get(w, "tools.0").Exists() {
		t.Fatalf("tools not carried: %s", w)
	}
	if tc := gjson.Get(w, "tool_choice.type").String(); tc != "none" {
		t.Errorf("tool_choice \"none\" becomes %s with the tools still defined, want {\"type\":\"none\"}", gjson.Get(w, "tool_choice").Raw)
	}
}

// findings/gemini-tool-choice-none.md (found by the go2lean M2 agent): a
// Gemini request that declares a function and sets function calling mode
// NONE.
func TestGo2LeanRepro_GeminiToolChoiceNoneStaysNone(t *testing.T) {
	contents := []*genai.Content{genai.NewContentFromText("What is the weather in Paris?", genai.RoleUser)}
	cfg := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "get_weather",
			Description: "Current weather for a city",
			Parameters: &genai.Schema{Type: genai.TypeObject, Properties: map[string]*genai.Schema{
				"city": {Type: genai.TypeString},
			}, Required: []string{"city"}},
		}}}},
		ToolConfig: &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeNone}},
	}
	chat := reproH3JSON(t, ConvertGoogleToOpenAIRequest("m", contents, cfg))
	if !gjson.Get(chat, "tools.0").Exists() {
		t.Fatalf("Gemini -> Chat: tools not carried: %s", chat)
	}
	if tc := gjson.Get(chat, "tool_choice").Raw; tc != `"none"` {
		t.Errorf("Gemini -> Chat: mode NONE becomes tool_choice %s with the tools still defined, want \"none\"", tc)
	}
	anth := reproH3JSON(t, ConvertGoogleToAnthropicRequest("m", contents, cfg))
	if !gjson.Get(anth, "tools.0").Exists() {
		t.Fatalf("Gemini -> Anthropic: tools not carried: %s", anth)
	}
	if tc := gjson.Get(anth, "tool_choice.type").String(); tc != "none" {
		t.Errorf("Gemini -> Anthropic: mode NONE becomes tool_choice %s with the tools still defined, want {\"type\":\"none\"}", gjson.Get(anth, "tool_choice").Raw)
	}
}
