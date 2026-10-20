# Finding 7: Cache reads counted twice in non-streaming Chat usage (new, not reported)

`ai.TokenUsage` defines `InputTokens` as the prompt *without* cache reads
(`CacheReadTokens` is separate; the prompt total is their sum), and
`usage.FromOpenAIChatCompletion` normalises OpenAI usage that way. The
OpenAI Chat passthrough's non-streaming handler does not use it:

```go
// internal/protocolserver/protocol_passthrough.go:35-40
inputTokens := int(response.Usage.PromptTokens)
cacheTokens := int(response.Usage.PromptTokensDetails.CachedTokens)
usage := protocol.NewTokenUsageWithCache(inputTokens, outputTokens, cacheTokens)
```

so the cached part of the prompt is recorded both as input and as cache
read. The streaming handler (`token.StreamTokenCounter.GetCounts`)
subtracts it. For the same upstream usage (`prompt_tokens` 1000,
`cached_tokens` 900) the gateway records:

| request | input | cache read |
|---|---|---|
| `stream: true` | 100 | 900 |
| non-streaming | 1000 | 900 |

The usage records, the dashboard's input totals and cache-hit share, the
per-service token stats used by the load balancer, and the OTel metrics
all take these numbers. With prompt caching a long session's input is
inflated by up to 10× on the non-streaming path, and cache write and
reasoning tokens are dropped on it (only three counts are copied).

Same pattern, by reading only: the Gemini paths
(`protocol_dispatch.go:272-280`, `stream/google_to_any.go:336,362,500,543`)
store `promptTokenCount` as input next to `cachedContentTokenCount`;
Gemini's `promptTokenCount` already includes the cached content.
`mcp_anthropic_v1_helper.go:182` and `toolengine/openai_chat_adapter.go:295`
build usage the same way.

How it was found: the usage direction of the spec work (every place that
builds a `TokenUsage` outside `internal/protocol/usage`), confirmed with
`go test` through the real gateway.

Repro: `repro/internal/protocolserver/` —
`TestGo2LeanRepro_NonStreamUsageExcludesCachedInput` sends one streaming
and one non-streaming request through a gateway built like the package's
own tests (`dual_provider_test.go`) and compares the recorded usage.

Fix direction: `usage.FromOpenAIChatCompletion(response.Usage)` in the
non-streaming handler (and the Google equivalent subtracting
`cachedContentTokenCount`).

Tracker search (2026-10-20) for `cached tokens usage`,
`cache input tokens`: earlier usage fixes (#1065, #1104, #1181, #1306)
cover the converters, not this handler.
