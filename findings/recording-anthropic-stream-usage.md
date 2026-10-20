# Finding 41: recordings of Anthropic streams lose input and cache usage (new, not reported; minor)

Severity: **low** (see [the findings table](README.md)).

Path: recording of a native Anthropic stream (Anthropic client -> Anthropic
provider, streaming, with recording on): the assembled response stored with
the recording.

Where (tingly-box `b788a3b`), `internal/protocol/assembler/anthropic_assembler.go`:

- `:48-51` (`RecordV1Event`) and `:79-82` (`RecordV1BetaEvent`):
  `message_start` keeps the id, type and role; its `usage` (input, cache
  creation, cache read) is ignored.
- `:59-71` and `:90-102`: a `message_delta` with output (or input) tokens
  replaces the usage with a new one built from the delta alone:
  `input_tokens` as the delta has it (usually absent, so 0),
  `cache_read_input_tokens` likewise, and `cache_creation_input_tokens` never
  copied (`:63-70`, `:94-101`).
- `Finish` (`:244`, usage `:264-272`) returns that usage. Used by
  `recording.AttachRecorderHooks` (`internal/recording/hook.go:20`, `:33-34`:
  `asm.Finish(model, 0, 0)` -> `recorder.SetAssembledResponse`), reached from
  `internal/protocolserver/stage_anthropic_adapter.go:115` and
  `mcp_anthropic_v1_helper.go:231`.

Trigger: an Anthropic stream whose `message_start` carries the input usage
and whose `message_delta` carries only `output_tokens` (what Anthropic's API
sends).

Observed: for `message_start` usage input 1200, cache creation 300, cache
read 5000, and `message_delta` output 42, the assembled usage is input 0,
cache creation 0, cache read 0, output 42.

Expected: input 1200, cache creation 300, cache read 5000, output 42: the
stream's final usage, each field from the last event that carries it.

Realism: low. Only the recording's assembled response is wrong; billing and
the usage clients see go through `usage.AnthropicAccumulator`, not this
assembler. A recording read to debug cost or caching shows no input and no
cache use for every Anthropic stream.

Status: **new, minor** (tracker search 2026-10-20 over the full text of all
issues and PRs for `message_start` with `usage`, `cache_creation`,
`recording`: nothing). Reproduces on main `bc460f6`.

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_RecordingAssemblerKeepsInputUsage`
(`repro/internal/protocol/assembler/zz_go2lean_repro_hunt4_test.go`).

## General property

For every Anthropic event stream (V1 or Beta) in which usage fields are
cumulative and a missing field means "unchanged", the usage of the
assembled message has, for each of input, cache creation, cache read and
output tokens, the value of the last event (message_start or message_delta)
that carries that field.

Target functions: `(*assembler.AnthropicStreamAssembler).RecordV1Event` and
`RecordV1BetaEvent`, read through `Finish`.
