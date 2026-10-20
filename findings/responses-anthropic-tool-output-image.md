# Responses -> Anthropic: images in tool output dropped (new, not reported)

Path: Responses client -> Anthropic provider (request conversion; streaming
or not).

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_responses_to_anthropic.go:331-353`
(`convertResponsesFunctionCallOutputToAnthropicBeta`). For an array output
it keeps only `input_text` items (`if item.OfInputText == nil { continue }`)
and concatenates them into one string; `input_image` (and `input_file`)
items are dropped. The Responses -> Chat conversion keeps the image, and
Anthropic -> Chat keeps tool_result images (issue #1606, cited in
`anthropic_view.go`).

Trigger: a `function_call_output` whose `output` is a list with an
`input_image` item: a screenshot or image-returning tool.

Observed: for output `[input_text "here", input_image (PNG data URL)]` the
Anthropic provider gets
`{"tool_use_id":"call_1","is_error":false,"content":[{"text":"here","type":"text"}],"type":"tool_result"}`.

Expected: the image as an image block inside `tool_result.content`, after
the text, as Anthropic tool results allow.

Realism: Codex and other Responses agents return images from tools
(screenshot and browser tools, MCP tools with image results) as
`input_image` items; on a Claude provider the model is told a screenshot
was taken and never sees it.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ToolOutputImageReachesAnthropic`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

## General property

For every Responses input, each `function_call_output` becomes a
`tool_result` whose content has, in order, one block per output item: a
text block for every text item and an image block (same data or URL) for
every image item.

Target function: `request.ConvertOpenAIResponsesToAnthropicBetaRequest`.
