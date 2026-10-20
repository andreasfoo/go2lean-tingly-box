# Finding 5: Refusal lost when streaming Responses -> Chat (new, not reported)

A Responses upstream answer whose message holds a `refusal` part:

- non-streaming (`nonstream.ConvertResponsesToOpenAIChat`): the Chat
  message has `refusal: "I can't help with that."`, finish `stop`;
- streaming (`responsesToChatConverter`,
  `internal/protocol/stream/openai_responses_to_chat_converter.go`): no
  content and no refusal at all, finish `stop`.

The stream converter has no case for `response.refusal.delta`, its
fallback for output sent only in `response.completed` copies
`output_text` parts only, and the chunk type `wire.ChatStreamDelta` has
no `refusal` field, so a streaming Chat client sees an empty answer.

Specs: `SpecResponsesToChatConverterProcessEventStream{Text,RefusalDelivered,Calls,Finish}Matches`
in `specs/internal/protocol/stream/`. Evidence so far is a direct go test
of `replayResponsesToChat` against `nonStreamedChat` on the answer above
(`refusal delivered: false`). The go2lean run of these specs checked no
trial yet: 1462 of 2000 random answers fail the shape precondition and
the rest stop at `fmt.Sprintf` in `newResponsesToChatConverter` (not
modelled). Related, not yet checked: the non-streaming path counts
`custom_tool_call` and `mcp_call` items as tool calls, the streaming
path only `function_call`.
