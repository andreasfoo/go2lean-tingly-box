# Finding 50: Anthropic -> Gemini: base64 images are encoded twice (found by hand while triaging the blind go2lean run; new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Anthropic client (V1 and Beta) -> Google-style provider (Gemini CLI,
Antigravity, a user-configured Google-style provider), request conversion,
streaming or not.

Where (tingly-box `b788a3b`):
`internal/protocol/request/anthropic_view.go:795-803`
(`convertAnthropicViewMessageToGoogle`, user branch) sets
`InlineData.Data = []byte(block.OfImage.Source.OfBase64.Data)` (`:801`):
the bytes of the base64 text, not the image. `genai.Blob.Data` is raw
bytes, which the SDK base64-encodes on the wire, so the image is encoded a
second time. The repo decodes correctly elsewhere: `imageURLToGoogleBlob`
(`internal/protocol/request/any_to_google.go:327-340`, used for images in
Chat tool messages at `:176`) base64-decodes before building the blob.
Reachable as in
`anthropic-gemini-tool-use-args.md` (#44).

Trigger: a user image block
`{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}`.

Observed: `inlineData.data` holds the 12 bytes `iVBORw0KGgo=`; on the wire
Gemini gets `"data":"aVZCT1J3MEtHZ289"`, the base64 of the base64 text.

Expected: the decoded bytes (`\x89PNG\r\n\x1a\n`), sent as
`"data":"iVBORw0KGgo="`.

Realism: medium. Every pasted screenshot or image from Claude Code (or any
Anthropic client) to a Gemini CLI or Antigravity provider reaches Gemini
as bytes that are not an image, labelled `image/png`, so the model
cannot see the image (whether Gemini then rejects the request or answers
without it was not checked against the live API).

Status: **new** (not reported; not yet checked against the tracker for an
existing issue).

How it was found: by hand, while triaging a spurious counterexample of the
blind go2lean run (see the [provenance note](README.md)). The blind spec
`SpecConvertAnthropicToGoogleRequestImageBytesAgree` states this property,
but go2lean's counterexample used data that is not valid base64, which the
model let decode while Go's reference decode fails and the spec skips the
input; so the refutation was spurious and did not show this bug. Reading
the code behind it did. The repro test was written afterwards with a
JSON-parsed request; with valid base64 the same spec catches the bug.

Evidence: `TestGo2LeanRepro_AnthropicToGeminiImageBytesDecoded`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Anthropic request and every user image block with a base64
source whose data is valid base64, the Google request holds an inlineData
part whose bytes equal the decoded data and whose MIME type is the media
type.

Target functions: `request.ConvertAnthropicToGoogleRequest` and
`request.ConvertAnthropicBetaToGoogleRequest` (via
`request.convertAnthropicViewMessageToGoogle`).
