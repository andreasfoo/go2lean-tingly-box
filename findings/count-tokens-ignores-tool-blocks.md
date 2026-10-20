# Finding 38: count_tokens estimate ignores tool_use, tool_result, document and image blocks (new, not reported)

Severity: **medium** (see [the findings table](README.md)).

Path: Anthropic client `POST /v1/messages/count_tokens` routed to a provider
without a native count endpoint: OpenAI and Google styles, and Anthropic
backends without count_tokens (Bedrock), where tingly-box estimates with
tiktoken.

Where (tingly-box `b788a3b`):

- `internal/protocol/token/anthropic.go:125` (`CountBetaTokensViaTiktoken`):
  the message loop (`:155-169`) counts the role and only `text`
  (`:160-162`) and `thinking` (`:163-165`) blocks. `tool_use` (its input),
  `tool_result` (its content), `document`, `image`, `search_result`,
  `redacted_thinking` and server-tool blocks count 0.
- The V1 function `CountTokensViaTiktoken` (`:52`, loop `:82-96`) has the
  same gap; it has no caller at `b788a3b`.
- Caller: `anthropicCountTokensViaTiktoken`
  (`internal/protocolserver/anthropic_count_tokens.go:163`), reached from
  `:137`, `:144` and `:149`.

Trigger: a count_tokens request whose messages contain a `tool_result` (or
`tool_use`, document, image) block.

Observed: adding a user message with a 20,800-byte `tool_result` to a
conversation changes the count from 8 to 9 (the role only); the preceding
`tool_use` also counts nothing.

Expected: the count grows by about the token count of the block's content
(thousands of tokens here), as it does for the same text in a `text` block.

Realism: medium. Agent conversations are mostly tool calls and tool
results, so the estimate misses most of the context; a client that calls
count_tokens to decide when to compact or truncate (to stay under the
model's window) under-counts and then overflows the window. Only the
estimate path is affected; native Anthropic count_tokens is exact.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `count_tokens`, `token estimat`: nothing). Reproduces on main
`bc460f6`.

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_CountTokensIncludesToolResult`
(`repro/internal/protocol/token/zz_go2lean_repro_hunt4_test.go`).

## General property

For every count_tokens request R and every content block b with textual
content t (the text of a text block, the JSON input of a tool_use, the text
content of a tool_result, the text of a document), adding b to a message of
R increases the estimate by at least the tokenizer count of t minus a small
constant; in particular the estimate is the same whether t is sent as a
text block or inside a tool_result.

Target function: `token.CountBetaTokensViaTiktoken` (and
`token.CountTokensViaTiktoken`).
