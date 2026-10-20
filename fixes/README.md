# Fixes for the high-severity findings, checked with go2lean

`protocol-fixes.diff` patches tingly-box b788a3b (12 files under
`internal/protocol`, +236/−98; spec files excluded). It applies with
`git apply` at the repository root.

## What it fixes

| area | change | findings |
|---|---|---|
| stream, Responses → Chat | tool_calls indexes count tool calls only; arguments that arrive only in `.done` are emitted; a call never announced by `output_item.added` is started there | #14, done-only arguments |
| stream, Responses → Anthropic | at most one open block; empty reasoning/summary deltas open no thinking block; done-only arguments reach `tool_use.input` | open-thinking finding |
| stream, Chat → Anthropic | `reasoning_content: null` or `""` opens no thinking block | reasoning-null finding |
| nonstream | `reasoning_content` becomes plain thinking text (not a JSON-quoted string), null gives no block, thinking comes first | reasoning-null finding |
| request | tool `input_schema` extra keywords (`$defs`, `additionalProperties`, …) kept for Chat and Responses, v1 and beta; Anthropic → Gemini `functionCall` args kept | #21, #44 |
| ops | the model transform keeps signed thinking on a tool-use turn while budget thinking stays on (`thinking.Disable{V1,Beta}ForUnsignedToolTurn`); the metadata transforms no longer panic | #36, #35 |
| metaid | `MetadataUserID.TryFix` returns an error; `Fix` keeps its API (panics through `TryFix`); `BuildMetadataUserID` and `FixMetadataUserID` use `TryFix` instead of `recover` | #35 |
| assembler | `ResponsesAssembler` records each `output_item.done` item and fills an empty terminal output (Codex) from them, in stream order | Codex non-stream function calls dropped |

One golden test changes its expectation:
`openai_responses_to_chat_golden_test.go` now expects the tool-call index to
count tool calls only.

## Verification

- `go vet ./internal/protocol/...` clean; `go test ./internal/protocol/...
  ./internal/protocolserver/...` pass.
- The 14 `TestGo2LeanRepro_*` tests (`repro/`): all fail before, all pass
  after.
- go2lean, the same specs before and after (budget 300 s, longer where
  noted):

| spec | before | after |
|---|---|---|
| ResponsesAssembler KeepsItems | confirmed | none in 966 trials (1800 s) |
| ChatToAnthropicBeta ThinkingIsText | confirmed | none in 2000 trials |
| BetaMetadataTransform no_panic | confirmed (panic) | none in 2000 trials (v1 and beta) |
| AnthropicBetaToResponses KeepsToolSchema | confirmed | none in 2000 trials |
| OpenAIToAnthropic OneOpenTextBlock | confirmed | none in 2000 trials |
| ResponsesToAnthropic OneOpenBlock | confirmed | none in 1000 trials (1800 s) |
| ResponsesToChat ArgumentsAnySchedule | confirmed | none in 3000 trials |
| ResponsesToChat ToolIndexesDense | confirmed | none in 3000 trials |
| BetaModelTransform ToolTurnThinking | confirmed | still confirmed: the spec is too broad (see below) |
| AnthropicBetaToOpenAIRequest KeepsToolSchema | confirmed | still confirmed: the spec is too broad (see below) |
| ChatToResponses TerminalUsage | confirmed | still confirmed: not fixed by this patch |

Every old counterexample of the 9 fixed specs satisfies its spec on the
fixed code.

The two "too broad" specs are confirmed after the fix on requests the APIs
reject: a `tool_result` answering an assistant turn with no `tool_use`, and
two tools with the same name. Narrower variants in `extra-specs/`
(`SpecFixwsBetaModelTransformToolUseTurnThinking`,
`SpecFixwsAnthropicBetaToOpenAIRequestKeepsToolSchemaUniqueNames`) are
confirmed on b788a3b (trial 12137 of 30000; trial 7) and find nothing on
the fixed code (30000 and 2000 trials).

Two go2lean limits shaped the patch: `recover` is not modelled (so the
recover-based metadata guard showed as a spurious panic, hence `TryFix`),
and neither is `sort.SliceStable` (the assembler keeps stream order rather
than sorting by output index).
