# Finding 77: Responses -> Anthropic requests send an empty text block for a user turn with only a file (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

`convertResponsesUserMessageToAnthropicBeta`
(`internal/protocol/request/openai_responses_to_anthropic.go:202-242`)
skips every part it cannot convert: `input_file` (`:231-232`) and an
`input_image` without `image_url`, i.e. by `file_id` (`:220-223`). When
nothing is left, it falls back to `NewBetaUserMessage(NewBetaTextBlock(""))`
(`:241`). The assistant side drops such blocks (`dropEmptyTextBlocks`,
`:156`, `:165`); the user side does not. Anthropic rejects an empty text block,
so the whole request fails instead of losing one attachment.

Path: a Responses client (Codex, the OpenAI Agents SDK) with an Anthropic
provider: `protocolserver/stage_openai_route.go:82`,
`stage/responsesbridge/bridge.go:53`.

Trigger: a Responses request with
`"input":[{"role":"user","content":[{"type":"input_file","filename":"report.pdf","file_data":"data:application/pdf;base64,JVBERi0xLjQK"}]}]`,
or with `{"type":"input_image","file_id":"file-abc123","detail":"auto"}` as
the only part.

Observed: `messages[0]` is
`{"role":"user","content":[{"type":"text","text":""}]}`; Anthropic answers
400 "text content blocks must be non-empty".

Expected: no empty text block: the PDF as a `document` block, or the
unsupported part skipped with the message left valid.

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecConvertOpenAIResponsesToAnthropicBetaRequestIssue1251NoEmptyText` (the
class of #1251 / #640, empty content sent upstream) was refuted, `go test`
confirmed the counterexample, and the repro test was written afterwards
with a PDF and a `file_id` image.

Repro: `TestGo2LeanRepro_ResponsesToAnthropicNoEmptyTextBlock`
(`repro/internal/protocol/request/zz_go2lean_repro_issues1_test.go`).
