# Finding 46: Chat -> Responses: developer messages dropped (found by go2lean, blind specs; new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Chat client -> Responses provider (the Codex OAuth provider, or an
OpenAI provider in per-model Responses mode), request conversion,
streaming or not.

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_chat_to_openai_responses.go`.
`ConvertChatToOpenAIResponses` (`:33-126`) moves system messages to
`instructions` and passes every other message, a developer message
included, to `ConvertChatMessagesToResponsesInput` (`:129-166`), whose
switch handles system (`:134`), user (`:137`), assistant (`:140`) and tool
(`:160`) and has no `OfDeveloper` case: the message vanishes. Callers:
`internal/protocol/stage/openaibridge/responses.go:47`,
`internal/protocol/transform/base.go:144`.

Trigger: a Chat request with a developer message:
`[{"role":"developer","content":"Answer in French."},{"role":"user","content":"What time is it?"}]`.

Observed: `{"input":[{"role":"user","content":"What time is it?","type":"message"}],…}`;
no `instructions`, and the developer text appears nowhere.

Expected: the developer text in `instructions`, or a `developer` input
message (Responses has the role).

Realism: medium, arguably high. Current OpenAI SDKs and agents send their
system prompt as a developer message (the guidance for o-series and gpt-5
models); a Chat client routed to a Responses-only provider loses its whole
prompt. The Chat -> Anthropic twin is `chat-anthropic-request-coverage.md`
(#18); `responses-developer-message-as-user.md` (#15) is the reverse
direction; `chat-responses-request-params.md` (#19) covers this function's
other fields.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `developer`: no issue for this direction; upstream's IR
round-trip registry, `internal/protocoltest/ir_roundtrip_gaps.go` from
#1889, lists dropped developer messages as a known gap of the Stage IR, a
different code path). Reproduces on main `bc460f6`.

How it was found: by go2lean, from a spec written blind (see the
[provenance note](README.md)):
`SpecConvertChatToOpenAIResponsesKeepsDeveloperMessages` was refuted, `go
test` confirmed the counterexample, and triage found no earlier finding
for it. The repro test was written afterwards with a JSON-parsed request.

Evidence: `TestGo2LeanRepro_ChatDeveloperMessageReachesResponses`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Chat request, the text of every system and developer message
appears in the Responses request, in `instructions` or in an input message
of role system or developer.

Target function: `request.ConvertChatToOpenAIResponses` (via
`request.ConvertChatMessagesToResponsesInput`).
