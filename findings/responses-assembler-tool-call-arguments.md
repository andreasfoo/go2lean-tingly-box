# Finding 42: `ResponsesAssembler.ToolCalls()` arguments always empty (new, not reported; latent)

Severity: **low** (see [the findings table](README.md)).

Path: the Responses stream assembler's tool-call accessor; no production
caller at `b788a3b` (`ToolCalls()` is used only by tests).

Where (tingly-box `b788a3b`):
`internal/protocol/assembler/openai_responses_assembler.go:105-109`
(`Accumulate`, `response.function_call_arguments.delta`) appends
`event.Arguments` (`:107`) to the pending call; on that event the wire
field holding the text is `delta` (`event.Delta`), and `arguments` is set
only on the `.done` event, which is ignored (`:111-114`). `ToolCalls()`
(`:322-336`) returns the accumulated, empty, string.

Trigger: any Responses stream with a function call whose arguments arrive
in `response.function_call_arguments.delta` events.

Observed: after the delta `{"cmd":"ls"}` (and its `.done`), `ToolCalls()[1].Arguments`
is `""`.

Expected: `{"cmd":"ls"}`.

Realism: low, latent. Nothing in the gateway reads `ToolCalls()` at
`b788a3b`; the first caller that does (for instance to fix finding 37 by
adding the calls to the assembled output) gets calls without arguments.

Status: **new, latent** (tracker search 2026-10-20 over the full text of all
issues and PRs for `ResponsesAssembler`, `ToolCalls`: nothing). Reproduces
on main `bc460f6`.

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ResponsesAssemblerToolCallArguments`
(`repro/internal/protocol/assembler/zz_go2lean_repro_hunt4_test.go`).

## General property

For every Responses event stream, `ToolCalls()[i].Arguments` equals the
concatenation of the `delta` fields of the
`response.function_call_arguments.delta` events of output index i, which is
the `arguments` of its `.done` event.

Target function: `(*assembler.ResponsesAssembler).ToolCalls` (after
`Accumulate`).
