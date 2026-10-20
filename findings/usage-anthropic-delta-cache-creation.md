# Anthropic stream usage: input depends on which fields message_delta repeats (new, not reported; low)

Path: usage from an Anthropic provider's stream: what tingly-box records
and what it reports to Chat and Responses clients converted from it.

Where (tingly-box `b788a3b`):
`internal/protocol/usage/accumulator.go:94-170`
(`(*AnthropicAccumulator).consumeRaw`, called once per event by `Consume`
and `ConsumeBeta`). Input tokens are *assigned* (`:107-125`: from
message_start, else from message_delta's `input_tokens`), while cache
creation is *added* to the input on every event that carries it
(`:149-157`). Users: `stage/upstream/anthropic.go:100` (recorded usage),
`stream/anthropic_to_openai_converter.go:70`,
`stream/anthropic_beta_to_openai_responses_converter.go:71`,
`stream/anthropic_passthrough.go`.

Trigger: message_start with `input_tokens` 10 and
`cache_creation_input_tokens` 100, and a message_delta whose (cumulative)
usage repeats only some of the input fields:

- only `cache_creation_input_tokens`: the creation is added a second time;
- only `input_tokens`: the input is overwritten with 10 and the creation is
  not added back.

Observed: `InputTokens` 210 in the first case; `InputTokens` 10 with
`CacheWriteTokens` 100 in the second.

Expected: 110 (input plus cache creation, each once) whatever subset of the
cumulative fields message_delta repeats; the cache writes a part of the
input.

Realism: low. message_delta usage is cumulative and its input fields are
optional; Anthropic's API repeats all of them or none, which gives the right
total. Anthropic-compatible providers that repeat only some of them get
input over- or under-counted, in billing records and in the usage clients
see.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_AccumulatorDeltaRepeatingCacheCreation` and
`TestGo2LeanRepro_AccumulatorDeltaRepeatingInput`
(`repro/internal/protocol/usage/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Anthropic event stream whose usage fields are cumulative (each
event's values are the totals so far, a missing field meaning "unchanged"),
the accumulated input tokens equal the last input tokens plus the last
cache-creation tokens seen, and the cache-write tokens are at most the
input tokens.

Target function: `(*usage.AnthropicAccumulator).ConsumeBeta` (and
`Consume`), read through `Result`.
