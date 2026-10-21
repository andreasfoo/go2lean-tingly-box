# Finding 78: Anthropic -> Gemini requests drop images inside tool_result (new, not reported; a documented known gap)

Severity: **medium** (see [the findings table](README.md)).

`convertAnthropicViewMessageToGoogle`
(`internal/protocol/request/anthropic_view.go:809-824`) turns a
`tool_result` into a `functionResponse` built from
`convertToolResultContent` (`request/anthropic_v1_to_openai.go:80`), which
keeps the text parts only. An image in the tool result never reaches
Gemini. PR #1613 (fix for #1606) carries tool images on every other
direction, Chat -> Gemini included, and lists this one as a follow-up;
`.design/multimodal-content.md:126-128` records it as a known gap. No
issue tracks it.

Path: an Anthropic V1 or Beta client with a Google-style provider:
`transform/base.go:218` and `:230`. Computer-use and screenshot tools, and
Claude Code reading an image file, return images this way.

Trigger: a history with
`{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"screenshot","input":{}}]}`
and `{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":[{"type":"text","text":"screenshot taken"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}]}]}`.

Observed: `{"functionResponse":{"name":"toolu_1","response":{"output":"screenshot taken"}}}`,
with no inline data anywhere in the request (V1 and Beta).

Expected: the image in `functionResponse.parts[].inlineData`, as the Chat
-> Gemini path sends it since #1613.

How it was found: by go2lean, from the issue-derived specs (see the
[provenance note](README.md)):
`SpecConvertAnthropicToGoogleRequestIssue1613ToolResultImages` and its Beta
twin were refuted, `go test` confirmed the counterexamples, and the repro
test was written afterwards with a screenshot tool result.

Repro: `TestGo2LeanRepro_AnthropicToGeminiToolResultImage`
(`repro/internal/protocol/request/zz_go2lean_repro_issues1_test.go`), V1
and Beta.
