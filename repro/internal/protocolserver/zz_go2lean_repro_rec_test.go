package protocolserver

// Reproduction for findings/recording-client-credentials.md: with recording
// on, internal/recording/recorder.go stores the inbound request's headers
// (headerToMap(c.Request.Header)) and URL (c.Request.URL.String()) as they
// came. On the gateway path the pre-transform recorder
// (protocolserver.TransformRecorder, StagePre) then replaces that request
// with one whose headers are an empty map but whose URL is still
// c.Request.URL.String(), query string included, and nothing redacts it
// before the record is written. The test runs the gateway with a rule that
// records (flags.recording "staged_request_response") and the default
// file-backed sink, sends an Anthropic request with the gateway token in
// x-api-key and Authorization, a cookie, and a key in the query string,
// reads the recording file back and fails while any of them is in it (on
// b788a3b and bc460f6 only the query-string key is).

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/tingly-dev/tingly-box/internal/client"
	"github.com/tingly-dev/tingly-box/internal/config"
	"github.com/tingly-dev/tingly-box/internal/loadbalance"
	"github.com/tingly-dev/tingly-box/internal/obs"
	"github.com/tingly-dev/tingly-box/internal/protocol"
	"github.com/tingly-dev/tingly-box/internal/routing"
	"github.com/tingly-dev/tingly-box/internal/typ"
)

const (
	go2leanRecClientKey   = "tingly-box-model-token-SECRET-1"
	go2leanRecQueryKey    = "query-key-SECRET-2"
	go2leanRecCookieValue = "session=cookie-SECRET-3"
)

// go2leanRecGateway starts the gateway with an Anthropic provider at
// upstreamURL, a recording rule for model "m" of the Anthropic scenario,
// and the file-backed sink the server builds (obs.NewSink(recordDir, mode)).
// It returns the gateway URL, the record directory and the sink.
func go2leanRecGateway(t *testing.T, upstreamURL string) (string, string, func() *obs.Sink) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	loadbalance.DefaultBreakerStore().Reset()
	cfg, err := config.NewConfig(config.WithConfigDir(t.TempDir()), config.WithDisableMigration(), config.WithDisableBuiltIn())
	require.NoError(t, err)
	require.NoError(t, cfg.AddProvider(&typ.Provider{UUID: "repro-rec", Name: "repro-rec", Enabled: true,
		AuthType: typ.AuthTypeAPIKey, Token: "provider-key", APIStyle: protocol.APIStyleAnthropic, APIBase: upstreamURL}))
	rule := typ.Rule{Scenario: typ.ScenarioAnthropic, RequestModel: "m", Active: true,
		Services: []*loadbalance.Service{{Provider: "repro-rec", Model: "claude-sonnet-4-5", Active: true}}}
	rule.Flags.Recording = string(obs.RecordModeStagedRequestResponse)
	require.NoError(t, cfg.AddRule(rule))

	recordDir := t.TempDir()
	var sink *obs.Sink
	hm := loadbalance.NewHealthMonitor(loadbalance.HealthMonitorConfig{ProbeEnabled: false})
	lb := NewLoadBalancer(cfg, routing.NewHealthFilter(hm))
	ph := NewHandler(ProtocolHandlerDeps{Config: cfg, ClientPool: client.NewClientPool(),
		RoutingSelector: routing.NewSimpleSelector(routing.NewServiceSelector(cfg, NewAffinityStore(0), lb)),
		LoadBalancer:    lb, HealthMonitor: hm,
		GetOrCreateScenarioSink: func(_ typ.RuleScenario, mode obs.RecordMode) *obs.Sink {
			if sink == nil {
				sink = obs.NewSink(recordDir, mode)
			}
			return sink
		}})
	engine := gin.New()
	ph.RegisterRoutes(engine, func(c *gin.Context) { c.Next() })
	gw := httptest.NewServer(engine)
	t.Cleanup(gw.Close)
	return gw.URL, recordDir, func() *obs.Sink { return sink }
}

// go2leanRecRead returns every record written under dir (.jsonl.gz
// session files, one JSON object per line).
func go2leanRecRead(t *testing.T, dir string) []map[string]any {
	t.Helper()
	var records []map[string]any
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".jsonl.gz") {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		zr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		sc := bufio.NewScanner(zr)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			var r map[string]any
			if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
				return err
			}
			records = append(records, r)
		}
		return sc.Err()
	})
	require.NoError(t, err)
	return records
}

func TestGo2LeanRepro_RecordingRedactsClientCredentials(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":1}}`)
	}))
	t.Cleanup(up.Close)
	gw, recordDir, sink := go2leanRecGateway(t, up.URL)

	req, err := http.NewRequest(http.MethodPost, gw+"/tingly/anthropic/v1/messages?key="+go2leanRecQueryKey,
		strings.NewReader(`{"model":"m","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("x-api-key", go2leanRecClientKey)
	req.Header.Set("Authorization", "Bearer "+go2leanRecClientKey)
	req.Header.Set("Cookie", go2leanRecCookieValue)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	require.NotNil(t, sink(), "recording sink not created: the rule's recording flag had no effect")
	sink().Close() // flushes the batch to disk
	records := go2leanRecRead(t, recordDir)
	require.NotEmpty(t, records, "no recording written under %s", recordDir)

	for _, r := range records {
		orig, _ := r["original_request"].(map[string]any)
		require.NotNil(t, orig, "record without original_request: %v", r)
		headers, _ := orig["headers"].(map[string]any)
		for name, v := range headers {
			s, _ := v.(string)
			if strings.Contains(s, "SECRET") {
				t.Errorf("recording keeps the client's %s header in clear: %q", name, s)
			}
		}
		if u, _ := orig["url"].(string); strings.Contains(u, "SECRET") {
			t.Errorf("recording keeps the client's query-string key in clear: url %q", u)
		}
	}
}
