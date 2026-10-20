# Gateway environment credentials sent to every upstream (new, not reported)

Both SDK clients are built with the SDKs' environment defaults
(`anthropic.NewClient` / `openai.NewClient` without
`option.WithoutEnvironmentDefaults()`), so credentials and headers from the
gateway's own process environment are attached to every upstream request
next to the provider's own key:

- `internal/client/anthropic.go:92` `newAnthropicClientWithTransport` sets
  `WithAPIKey(provider token)` (header `X-Api-Key`). The SDK defaults
  (`libs/anthropic-sdk-go/client.go:53` `DefaultClientOptions`) add
  `ANTHROPIC_AUTH_TOKEN` as `Authorization: Bearer …`, which the provider
  key does not override because it is a different header. With
  `ANTHROPIC_API_KEY` set instead, Bedrock / Vertex providers (which skip
  `WithAPIKey`) get it as `X-Api-Key`.
- `internal/client/openai.go:82` `newOpenAIClientWithTransport`: the
  defaults add `OPENAI_ORG_ID` and `OPENAI_PROJECT_ID` as
  `OpenAI-Organization` / `OpenAI-Project` (and `OPENAI_CUSTOM_HEADERS`) to
  every OpenAI-compatible provider.

Why it matters: tingly-box is set up as Claude Code's gateway by exporting
`ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` (the gateway token). A
gateway started from that shell, or from a profile that exports them,
forwards that token to every third-party Anthropic-style provider. Some
Anthropic-compatible endpoints read `Authorization` before `X-Api-Key`, so
the request is also authenticated with the wrong credential. An
`OpenAI-Organization` header meant for api.openai.com goes to every other
vendor too. The transports already take care not to inherit the
environment's proxy settings (`transport_pool.go`); credentials are not
treated the same way.

How it was found: by reading the credential paths (the credentials
direction of the spec work), confirmed with `go test`.

Repro: `repro/internal/client/` —
`TestGo2LeanRepro_EnvAuthTokenNotSentUpstream` (an Anthropic provider
with key `provider-key`, `ANTHROPIC_AUTH_TOKEN=gateway-env-token` in the
environment: the upstream receives `Authorization: Bearer
gateway-env-token` and `X-Api-Key: provider-key`) and
`TestGo2LeanRepro_EnvOpenAIOrgNotSentUpstream`.

Fix direction: pass `option.WithoutEnvironmentDefaults()` (both SDKs have
it) in the two constructors and any other `NewClient` call on provider
credentials.

Tracker search (2026-10-20) for `ANTHROPIC_AUTH_TOKEN`, `OPENAI_ORG_ID`,
`environment variable`: nothing about upstream requests.
