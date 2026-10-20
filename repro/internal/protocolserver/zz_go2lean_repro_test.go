package protocolserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/tingly-dev/tingly-box/internal/client"
	"github.com/tingly-dev/tingly-box/internal/config"
	"github.com/tingly-dev/tingly-box/internal/loadbalance"
	"github.com/tingly-dev/tingly-box/internal/protocol"
	"github.com/tingly-dev/tingly-box/internal/routing"
	"github.com/tingly-dev/tingly-box/internal/typ"
)

// Finding usage-nonstream-cached-input: on the OpenAI Chat passthrough the
// non-streaming handler records prompt_tokens as InputTokens and
// cached_tokens as CacheReadTokens, so cache hits are counted twice
// (ai.TokenUsage: InputTokens excludes cache reads). The streaming handler
// subtracts them, so the same upstream usage is recorded differently.

const go2leanReproUsage = `{"prompt_tokens":1000,"completion_tokens":10,"total_tokens":1010,"prompt_tokens_details":{"cached_tokens":900}}`

func go2leanReproGateway(t *testing.T) string {
	t.Helper()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), `"stream":true`) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, `data: {"id":"c","object":"chat.completion.chunk","created":1,"model":"up","choices":[{"index":0,"delta":{"role":"assistant","content":"hi"},"finish_reason":null}]}`+"\n\n")
			_, _ = io.WriteString(w, `data: {"id":"c","object":"chat.completion.chunk","created":1,"model":"up","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":`+go2leanReproUsage+"}\n\n")
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"c","object":"chat.completion","created":1,"model":"up","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":`+go2leanReproUsage+`}`)
	}))
	t.Cleanup(up.Close)

	loadbalance.DefaultBreakerStore().Reset()
	cfg, err := config.NewConfig(config.WithConfigDir(t.TempDir()), config.WithDisableMigration(), config.WithDisableBuiltIn())
	require.NoError(t, err)
	require.NoError(t, cfg.AddProvider(&typ.Provider{UUID: "repro-p", Name: "repro-p", Enabled: true,
		AuthType: typ.AuthTypeAPIKey, Token: "k", APIStyle: protocol.APIStyleOpenAI, APIBase: up.URL + "/v1"}))
	require.NoError(t, cfg.AddRule(typ.Rule{Scenario: typ.ScenarioOpenAI, RequestModel: "m", Active: true,
		Services: []*loadbalance.Service{{Provider: "repro-p", Model: "up", Active: true}}}))
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

// recordedInput sends body through the gateway and returns the input and
// cache-read counts trackUsageWithTokenUsage logged for it.
func recordedInput(t *testing.T, gw, body string) (in, cache any) {
	t.Helper()
	hook := logtest.NewGlobal()
	defer hook.Reset()
	level := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(level)
	resp, err := http.Post(gw+"/tingly/openai/v1/chat/completions", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "token usage recorded") {
			return e.Data["input_tokens"], e.Data["cache_tokens"]
		}
	}
	t.Fatal("no usage recorded")
	return nil, nil
}

func TestGo2LeanRepro_NonStreamUsageExcludesCachedInput(t *testing.T) {
	gw := go2leanReproGateway(t)
	sIn, sCache := recordedInput(t, gw, `{"model":"m","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"hi"}]}`)
	nIn, nCache := recordedInput(t, gw, `{"model":"m","messages":[{"role":"user","content":"hi"}]}`)
	t.Logf("prompt_tokens=1000 cached_tokens=900: stream records input=%v cache=%v, non-stream records input=%v cache=%v", sIn, sCache, nIn, nCache)
	if nIn != 100 {
		t.Errorf("non-stream recorded input_tokens=%v, want 100 (prompt 1000 minus 900 cache reads, as the stream path records %v)", nIn, sIn)
	}
}
