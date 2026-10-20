# Finding 25: PDF, document and file content dropped from requests (new, not reported)

Severity: **high** (see [the findings table](README.md)).

Path: Anthropic client -> Chat or Responses provider, Chat client ->
Anthropic or Responses provider (request conversion; streaming or not).

Where (tingly-box `b788a3b`), `internal/protocol/request/`:

- `anthropic_view.go:76-126` (`viewAnthropicBetaBlock`): the switch has
  cases for text, thinking, redacted_thinking, tool_use, tool_result and
  image, none for `OfDocument`; a document block becomes an empty block and
  is dropped. The Anthropic -> Chat and Anthropic -> Responses converters
  both go through this view (`anthropic_beta_to_responses.go:113-118`); the
  tool_result content switch in the same function keeps text and images
  only.
- `openai_to_anthropic.go:43-57` (`ConvertOpenAIToAnthropicRequest`): user
  content parts are converted for `text` and `image_url` only; a `file` part
  is dropped.
- `openai_chat_to_openai_responses.go:197-218`
  (`convertChatUserMessageToResponses`): the same, `text` and `image_url`
  only.

Trigger: an Anthropic `document` block (a PDF as base64 or URL, or plain
text), or a Chat `{"type":"file","file":{"file_data":…}}` part.

Observed: for a user message `[document (PDF), text "Summarize this PDF."]`
the Chat provider gets
`[{"content":[{"text":"Summarize this PDF.","type":"text"}],"role":"user"}]`
and the Responses provider the same with `input_text`; for a Chat file part
the Anthropic and Responses providers get only the text.

Expected: the document reaches the provider: a Chat `file` part, a
Responses `input_file` part, an Anthropic `document` block, with the same
data (each of these providers accepts PDFs in that form).

Realism: high. Sending a PDF is a routine use of all three APIs (Claude
Code reads PDFs into document blocks; the OpenAI SDKs send files as `file`
/ `input_file` parts). Nothing fails: the model answers without the
document, typically by making up its content or saying it cannot see one.

Status: **new** (no issue found by the round-3 tracker search, 2026-10-20).

How it was found: by hand (code reading, round 3), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_AnthropicDocumentBlockCarried` and
`TestGo2LeanRepro_ChatFilePartCarried`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt3_test.go`).

## General property

For every request, every document or file content part of a message (an
Anthropic `document` block, a Chat `file` part) is in the converted request
as the target protocol's document or file part, with the same data (or URL
or file id) and media type.

Target functions: `request.ConvertAnthropicBetaToOpenAIRequest`,
`request.ConvertAnthropicBetaToResponsesRequest`,
`request.ConvertOpenAIToAnthropicRequest`,
`request.ConvertChatToOpenAIResponses`.
