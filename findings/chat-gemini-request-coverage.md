# Finding 47: Chat -> Gemini requests drop max_completion_tokens, developer and array-form system text, and user images (found by go2lean, blind specs; new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Chat client -> Google-style provider (Gemini CLI, Antigravity, a
user-configured Google-style provider), request conversion, streaming or
not. The gateway calls the converter from `convertToGoogle`
(`internal/protocol/transform/base.go:240-250`) with a default of 4096
tokens.

Where (tingly-box `b788a3b`):
`internal/protocol/request/any_to_google.go`, `ConvertOpenAIToGoogleRequest`
(`:60-215`):

- `:66-70` reads only `max_tokens`; otherwise `maxOutputTokens` is the
  default (4096). `max_completion_tokens` is never read.
- `:89-94` takes system text only in string form; an array of text parts
  gives nothing.
- the message switch (`:89`, `:96`, `:121`, `:156`) has no developer case,
  so developer messages are dropped.
- `:109-114` converts only text parts of a user message ("Handle images or
  other content types if needed"); `image_url` parts are dropped.

Trigger: a Chat request with `max_completion_tokens: 16000`, a system
message `[{"type":"text","text":"You are a travel agent."}]`, a developer
message `"Answer in French."`, and a user message with a text part and a
data-URL `image_url` part.

Observed: `maxOutputTokens: 4096`, `systemInstruction: null`, the user
content holding only the text part.

Expected: `maxOutputTokens: 16000`; both texts in the systemInstruction;
the image as an inlineData part (the decoded bytes and MIME type of the
data URL).

Realism: medium. Current OpenAI SDK clients send `max_completion_tokens`
(`max_tokens` is deprecated for reasoning models) and developer or
array-form system messages; vision clients send `image_url`. Answers are
cut at 4096 tokens, the prompt is lost and images silently disappear.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `gemini` with `system`, `developer`, `image`,
`max_completion_tokens`: nothing; #858, merged, restored images on the other
converters). Reproduces on main `bc460f6`.

How it was found: by go2lean, from specs written blind (see the
[provenance note](README.md)):
`SpecConvertOpenAIToGoogleRequestKeepsMaxCompletionTokens`,
`SpecConvertOpenAIToGoogleRequestKeepsSystemAndDeveloperText` and
`SpecConvertOpenAIToGoogleRequestKeepsUserImages` were refuted, `go test`
confirmed the counterexamples, and triage found no earlier finding for
them. The repro tests were written afterwards with JSON-parsed requests.

Evidence: `TestGo2LeanRepro_ChatToGeminiKeepsMaxCompletionTokens`,
`TestGo2LeanRepro_ChatToGeminiKeepsDeveloperAndSystemText` and
`TestGo2LeanRepro_ChatToGeminiKeepsUserImages`
(`repro/internal/protocol/request/zz_go2lean_repro_blind_test.go`).

## General property

For every Chat request: `maxOutputTokens` equals `max_completion_tokens`
when set (else `max_tokens`, else the default); the text of every system
and developer message, in string or array form, appears in the
systemInstruction; every user image part reaches the Gemini contents.

Target function: `request.ConvertOpenAIToGoogleRequest`.
