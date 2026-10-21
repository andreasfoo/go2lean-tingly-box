# Finding 70: Responses -> Chat streams drop reasoning summaries and text (new, not reported)

Severity: **low** (see [the findings table](README.md)); the trigger is
narrow.

`responsesToChatConverter.processEvent`
(`internal/protocol/stream/openai_responses_to_chat_converter.go:106-228`)
has no case for the `response.reasoning_summary_text.*` or
`response.reasoning_text.*` events, so no `reasoning_content` reaches the
Chat client. `reasoning_content` is tingly-box's own Chat convention for
reasoning: the Anthropic -> Chat stream emits it for thinking.

Path: a Chat client with a Responses provider, streaming:
`protocolserver/protocol_cross.go:83`, `stage/openaibridge/responses_stream.go:24`.
The Chat -> Responses request never sets `reasoning.summary`
(`request/openai_chat_to_openai_responses.go:111-116` forwards only the
effort), so OpenAI sends no summary; only upstreams that emit reasoning
events on their own (for example gpt-oss on vLLM, with `reasoning_text`)
are affected.

Trigger: Responses events with a reasoning item whose summary streams as
`response.reasoning_summary_text.delta` "**Greeting** The user says hi.",
then a message "Hello!".

Observed: no Chat chunk carries `reasoning_content`.

Expected: `delta.reasoning_content` "**Greeting** The user says hi.", as
the Anthropic -> Chat stream gives for thinking.

How it was found: with the clients' own SDKs as the oracle, as for
[finding 62](gemini-anthropic-stream-block-index.md).

Repro: `TestGo2LeanRepro_ResponsesToChatStreamKeepsReasoning`
(`repro/internal/protocol/stream/zz_go2lean_repro_sdk_test.go`).
