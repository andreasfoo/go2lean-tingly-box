# Finding 15: Responses developer messages become user messages on a Chat provider (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Responses client -> Chat provider (request conversion; streaming or
not).

Where (tingly-box `b788a3b`):
`internal/protocol/request/openai_responses_to_chat.go:256-259`
(`createMessageFromResponsesContent`): a message whose content is text parts
only (no image, no cache breakpoint) goes to `createMessage(role, text)`
(`:339-350`), which knows `system`, `user` and `assistant` and sends every
other role, `developer` included, to `openai.UserMessage`. The other branch
of the same function (images or cache breakpoints) maps `developer` to
`openai.DeveloperMessage` (`:300`), so the role depends on the content form.

Trigger: a Responses `input` message item with `"role": "developer"` and
`input_text` content.

Observed: the upstream gets
`{"messages":[{"content":"Never run rm -rf.","role":"user"},{"content":"hi","role":"user"}],…}`.

Expected: `{"role":"developer","content":"Never run rm -rf."}` (or a system
message for upstreams without the developer role), as the image/cache branch
already does.

Realism: Codex sends its permission and sandbox instructions
(`<permissions instructions>…`) and collaboration-mode instructions as
developer messages with `input_text` content in `input` on every turn. On a
Chat provider they arrive with the authority of user text, next to whatever
the user (or a tool result quoted by the user) says, so the model can weigh
them like any user claim.

Status: **new** (tracker search 2026-10-20 for `developer`, `developer
role`: nothing about this; #1921 is system messages dropped on the
Anthropic paths).

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_ResponsesDeveloperMessageStaysDeveloper`
(`repro/internal/protocol/request/zz_go2lean_repro_hunt2_test.go`) and
`TestGo2LeanRepro_GatewayDeveloperRoleReachesChatUpstream`
(`repro/internal/protocolserver/zz_go2lean_repro_hunt2_test.go`, the request
the upstream receives).

Fix direction: a `developer` case in `createMessage`.

## General property

For every Responses input message item, the role of the Chat message it
converts to is the item's role (`developer` and `system` stay instruction
roles; only `user` becomes user), independent of the form of its content
(string, text parts, images, cache breakpoints).

Target function: `request.ConvertResponsesInputToMessages` (via
`request.ConvertOpenAIResponsesToChat`).
