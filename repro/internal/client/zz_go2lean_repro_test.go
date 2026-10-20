package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"

	"github.com/tingly-dev/tingly-box/internal/protocol"
	"github.com/tingly-dev/tingly-box/internal/typ"
)

// Finding env-credentials-upstream: both SDK clients are built without
// WithoutEnvironmentDefaults, so credentials and headers from the gateway's
// own process environment are attached to every upstream request, next to
// the provider's key. A Claude Code user who exports ANTHROPIC_AUTH_TOKEN
// (often the gateway's own token) and starts tingly-box from that shell
// sends it as "Authorization: Bearer" to every Anthropic-style provider.

// headerCapture serves 500 to every request and keeps the request headers.
func headerCapture(t *testing.T) (*httptest.Server, func() []http.Header) {
	t.Helper()
	var mu sync.Mutex
	var got []http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got = append(got, r.Header.Clone())
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []http.Header {
		mu.Lock()
		defer mu.Unlock()
		return got
	}
}

func TestGo2LeanRepro_EnvAuthTokenNotSentUpstream(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "gateway-env-token")
	srv, headers := headerCapture(t)
	p := &typ.Provider{UUID: "repro-anthropic", Name: "repro-anthropic", Enabled: true,
		AuthType: typ.AuthTypeAPIKey, Token: "provider-key",
		APIStyle: protocol.APIStyleAnthropic, APIBase: srv.URL}
	c, err := NewAnthropicClient(p, "m", typ.SessionID{})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.MessagesNew(context.Background(), &anthropic.MessageNewParams{
		Model: "m", MaxTokens: 1,
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("hi"))},
	})
	got := headers()
	if len(got) == 0 {
		t.Fatal("no upstream request")
	}
	if a := got[0].Get("Authorization"); a != "" {
		t.Errorf("upstream got Authorization %q from the gateway's environment (x-api-key %q)", a, got[0].Get("X-Api-Key"))
	}
}

func TestGo2LeanRepro_EnvOpenAIOrgNotSentUpstream(t *testing.T) {
	t.Setenv("OPENAI_ORG_ID", "org-from-env")
	t.Setenv("OPENAI_PROJECT_ID", "proj-from-env")
	srv, headers := headerCapture(t)
	p := &typ.Provider{UUID: "repro-openai", Name: "repro-openai", Enabled: true,
		AuthType: typ.AuthTypeAPIKey, Token: "provider-key",
		APIStyle: protocol.APIStyleOpenAI, APIBase: srv.URL}
	c, err := NewOpenAIClient(p, "m", typ.SessionID{})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.Client().Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model:    "m",
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("hi")},
	})
	got := headers()
	if len(got) == 0 {
		t.Fatal("no upstream request")
	}
	if o, p := got[0].Get("OpenAI-Organization"), got[0].Get("OpenAI-Project"); o != "" || p != "" {
		t.Errorf("upstream got OpenAI-Organization %q / OpenAI-Project %q from the gateway's environment", o, p)
	}
}
