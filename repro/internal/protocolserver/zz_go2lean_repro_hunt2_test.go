package protocolserver

// Reproductions through the real gateway (routes, routing, transforms,
// upstream client) for findings found by hand (code reading, round 2). The
// gateway is built like the package's own tests (dual_provider_test.go);
// each test asserts what the client or the upstream should see and fails
// while the bug is there.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/tingly-dev/tingly-box/ai"
	"github.com/tingly-dev/tingly-box/internal/client"
	"github.com/tingly-dev/tingly-box/internal/config"
	"github.com/tingly-dev/tingly-box/internal/loadbalance"
	"github.com/tingly-dev/tingly-box/internal/protocol"
	"github.com/tingly-dev/tingly-box/internal/routing"
	"github.com/tingly-dev/tingly-box/internal/typ"
)

// go2leanH2Upstream answers every request with status, contentType and
// body, and returns a func giving the last request body it received.
func go2leanH2Upstream(t *testing.T, status int, contentType, body string) (string, func() string) {
	t.Helper()
	var mu sync.Mutex
	var last string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		last = string(b)
		mu.Unlock()
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() string { mu.Lock(); defer mu.Unlock(); return last }
}

// go2leanH2Gateway starts the gateway with one provider at upstreamURL and
// one rule mapping model "m" of scenario to upstreamModel; it returns the
// gateway's base URL.
func go2leanH2Gateway(t *testing.T, upstreamURL string, apiStyle protocol.APIStyle, mode ai.OpenAIEndpointMode, scenario typ.RuleScenario, upstreamModel string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	loadbalance.DefaultBreakerStore().Reset()
	cfg, err := config.NewConfig(config.WithConfigDir(t.TempDir()), config.WithDisableMigration(), config.WithDisableBuiltIn())
	require.NoError(t, err)
	base := upstreamURL + "/v1"
	if apiStyle == protocol.APIStyleAnthropic {
		base = upstreamURL
	}
	require.NoError(t, cfg.AddProvider(&typ.Provider{UUID: "repro-h2", Name: "repro-h2", Enabled: true,
		AuthType: typ.AuthTypeAPIKey, Token: "k", APIStyle: apiStyle, APIBase: base, OpenAIEndpointMode: mode}))
	require.NoError(t, cfg.AddRule(typ.Rule{Scenario: scenario, RequestModel: "m", Active: true,
		Services: []*loadbalance.Service{{Provider: "repro-h2", Model: upstreamModel, Active: true}}}))
	hm := loadbalance.NewHealthMonitor(loadbalance.HealthMonitorConfig{ProbeEnabled: false})
	lb := NewLoadBalancer(cfg, routing.NewHealthFilter(hm))
	ph := NewHandler(ProtocolHandlerDeps{Config: cfg, ClientPool: client.NewClientPool(),
		RoutingSelector: routing.NewSimpleSelector(routing.NewServiceSelector(cfg, NewAffinityStore(0), lb)),
		LoadBalancer:    lb, HealthMonitor: hm})
	engine := gin.New()
	ph.RegisterRoutes(engine, func(c *gin.Context) { c.Next() })
	gw := httptest.NewServer(engine)
	t.Cleanup(gw.Close)
	return gw.URL
}

// go2leanH2Post posts body to gw+path and returns the status and body.
func go2leanH2Post(t *testing.T, gw, path, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(gw+path, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// findings/responses-stream-trailing-usage.md: Responses client, Chat
// upstream, streaming.
func TestGo2LeanRepro_GatewayResponsesStreamGetsUsage(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		`data: {"id":"c1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[],"usage":{"prompt_tokens":1200,"completion_tokens":7,"total_tokens":1207,"prompt_tokens_details":{"cached_tokens":1024}}}`,
		`data: [DONE]`, ``, ``}, "\n\n")
	up, upstreamBody := go2leanH2Upstream(t, 200, "text/event-stream", sse)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleOpenAI, "", typ.ScenarioOpenAI, "gpt-4o")
	code, body := go2leanH2Post(t, gw, "/tingly/openai/v1/responses", `{"model":"m","stream":true,"input":"hi"}`)
	require.Equal(t, 200, code, body)
	if !strings.Contains(upstreamBody(), `"include_usage":true`) {
		t.Errorf("streaming Chat request sent without stream_options.include_usage, so an OpenAI upstream sends no usage: %s", upstreamBody())
	}
	completed := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data:") && strings.Contains(line, `"response.completed"`) {
			completed = line
		}
	}
	if !strings.Contains(completed, `"input_tokens":1200`) {
		t.Errorf("response.completed does not carry the upstream usage (input_tokens 1200, sent after finish_reason): %s", completed)
	}
}

// findings/responses-developer-message-as-user.md: Responses client, Chat
// upstream.
func TestGo2LeanRepro_GatewayDeveloperRoleReachesChatUpstream(t *testing.T) {
	up, upstreamBody := go2leanH2Upstream(t, 200, "application/json",
		`{"id":"x","object":"chat.completion","created":1,"model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleOpenAI, "", typ.ScenarioOpenAI, "gpt-4o")
	code, body := go2leanH2Post(t, gw, "/tingly/openai/v1/responses", `{"model":"m","input":[
	  {"type":"message","role":"developer","content":[{"type":"input_text","text":"Never run rm -rf."}]},
	  {"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}]}`)
	require.Equal(t, 200, code, body)
	if strings.Contains(upstreamBody(), `{"content":"Never run rm -rf.","role":"user"}`) {
		t.Errorf("developer instruction sent upstream as a user message: %s", upstreamBody())
	}
}

// findings/chat-tool-call-empty-id.md: Chat upstream returning a tool call
// without id, to an Anthropic and to a Responses client.
func TestGo2LeanRepro_GatewayToolCallWithoutIDGetsID(t *testing.T) {
	answer := `{"id":"x","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`
	up, _ := go2leanH2Upstream(t, 200, "application/json", answer)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleOpenAI, "", typ.ScenarioAnthropic, "m")
	code, body := go2leanH2Post(t, gw, "/tingly/anthropic/v1/messages", `{"model":"m","max_tokens":100,"messages":[{"role":"user","content":"weather?"}],
	  "tools":[{"name":"get_weather","input_schema":{"type":"object","properties":{"city":{"type":"string"}}}}]}`)
	require.Equal(t, 200, code, body)
	if strings.Contains(body, `"id":""`) {
		t.Errorf("Anthropic client gets a tool_use with id \"\": %s", body)
	}
	up2, _ := go2leanH2Upstream(t, 200, "application/json", answer)
	gw2 := go2leanH2Gateway(t, up2, protocol.APIStyleOpenAI, "", typ.ScenarioOpenAI, "m")
	code, body = go2leanH2Post(t, gw2, "/tingly/openai/v1/responses", `{"model":"m","input":"weather?",
	  "tools":[{"type":"function","name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}]}`)
	require.Equal(t, 200, code, body)
	if !strings.Contains(body, `"call_id"`) {
		t.Errorf("Responses client gets a function_call without call_id: %s", body)
	}
}

// findings/chat-anthropic-request-coverage.md: Chat client, Anthropic
// upstream.
func TestGo2LeanRepro_GatewayAnthropicUpstreamGetsChatParams(t *testing.T) {
	up, upstreamBody := go2leanH2Upstream(t, 200, "application/json",
		`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":1}}`)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleAnthropic, "", typ.ScenarioOpenAI, "claude-sonnet-4-5")
	code, body := go2leanH2Post(t, gw, "/tingly/openai/v1/chat/completions", `{"model":"m",
	  "messages":[{"role":"developer","content":"Answer in French."},{"role":"user","content":"What time is it?"}],
	  "tools":[{"type":"function","function":{"name":"get_time","description":"current time"}},
	           {"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}],
	  "tool_choice":"required"}`)
	require.Equal(t, 200, code, body)
	sent := upstreamBody()
	if !strings.Contains(sent, "Answer in French.") {
		t.Errorf("developer message not sent upstream")
	}
	if !strings.Contains(sent, `"get_time"`) {
		t.Errorf("tool without parameters not sent upstream")
	}
	if !strings.Contains(sent, `"tool_choice":{"type":"any"}`) {
		t.Errorf("tool_choice \"required\" not sent as {\"type\":\"any\"}")
	}
	if t.Failed() {
		t.Logf("upstream request: %s", sent)
	}
}

// findings/chat-responses-request-params.md: Chat client, Responses
// upstream.
func TestGo2LeanRepro_GatewayResponsesUpstreamGetsChatParams(t *testing.T) {
	up, upstreamBody := go2leanH2Upstream(t, 200, "application/json",
		`{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"gpt-4.1","output":[{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"ok","annotations":[]}]}],"usage":{"input_tokens":3,"output_tokens":1,"total_tokens":4}}`)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleOpenAI, ai.EndpointModeResponses, typ.ScenarioOpenAI, "gpt-4.1")
	code, body := go2leanH2Post(t, gw, "/tingly/openai/v1/chat/completions", `{"model":"m","messages":[{"role":"user","content":"hi"}],
	  "temperature":0,"max_completion_tokens":50,"parallel_tool_calls":false,
	  "tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}]}`)
	require.Equal(t, 200, code, body)
	sent := upstreamBody()
	for _, want := range []string{`"max_output_tokens":50`, `"temperature":0`, `"parallel_tool_calls":false`} {
		if !strings.Contains(sent, want) {
			t.Errorf("upstream request lacks %s", want)
		}
	}
	if t.Failed() {
		t.Logf("upstream request: %s", sent)
	}
}

// findings/upstream-errors-not-translated.md: Anthropic client, Chat
// upstream answering 429.
func TestGo2LeanRepro_GatewayErrorIsAnthropicShaped(t *testing.T) {
	up, _ := go2leanH2Upstream(t, 429, "application/json",
		`{"error":{"message":"Rate limit reached for gpt-4o","type":"requests","code":"rate_limit_exceeded"}}`)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleOpenAI, "", typ.ScenarioAnthropic, "up-model")
	code, body := go2leanH2Post(t, gw, "/tingly/anthropic/v1/messages", `{"model":"m","max_tokens":10,"messages":[{"role":"user","content":"hi"}]}`)
	if !strings.Contains(body, `"type":"error"`) || !strings.Contains(body, `"type":"rate_limit_error"`) {
		t.Errorf("upstream 429 reaches the Anthropic client as %d %s, want an Anthropic error {\"type\":\"error\",\"error\":{\"type\":\"rate_limit_error\",...}}", code, body)
	}
}

// findings/upstream-errors-not-translated.md: Responses client, Chat
// upstream answering 400 context_length_exceeded (Codex compacts on it).
func TestGo2LeanRepro_GatewayErrorCodeReachesResponsesClient(t *testing.T) {
	up, _ := go2leanH2Upstream(t, 400, "application/json",
		`{"error":{"message":"This model's maximum context length is 128000 tokens","type":"invalid_request_error","code":"context_length_exceeded"}}`)
	gw := go2leanH2Gateway(t, up, protocol.APIStyleOpenAI, "", typ.ScenarioOpenAI, "up-model")
	code, body := go2leanH2Post(t, gw, "/tingly/openai/v1/responses", `{"model":"m","input":"hi"}`)
	if !strings.Contains(body, `"code":"context_length_exceeded"`) {
		t.Errorf("upstream error code context_length_exceeded lost; the Responses client gets %d %s", code, body)
	}
}
