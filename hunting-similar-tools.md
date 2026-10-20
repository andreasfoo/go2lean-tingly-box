# Bug classes in similar LLM gateways (for checking tingly-box)

Sources: GitHub issue trackers of BerriAI/litellm (LL), musistudio/claude-code-router (CCR),
QuantumNous/new-api (NA), farion1231/cc-switch (CCS), fuergaosi233/claude-code-proxy (FCP),
1rgs/claude-code-proxy (RCP). songquanpeng/one-api had almost no relevant issues (it is
largely unmaintained; new-api is its active fork). Triggers below were read from the issue
bodies (survey date 2026-10-20). Notation: A = Anthropic Messages, C = OpenAI Chat
Completions, R = OpenAI Responses.

Each entry gives the trigger, the direction or subsystem, the evidence, and a property you
can check against a Go function.

---

## A. Stream conversion C -> A (Chat SSE to Anthropic SSE)

### 1. Parallel tool calls merged into one tool_use block (tool_call.index ignored)
- **Trigger:** One upstream Chat chunk (or consecutive chunks) carries several `delta.tool_calls` with index 0..N. The adapter only looks at `tool_calls[0]` and concatenates every call's arguments into one `input_json_delta` stream under the first call's id and name.
- **Direction:** stream C->A
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/44029
- **Property:** The number of `tool_use` blocks equals the number of distinct upstream `tool_calls[].index` values. For each index, the concatenated `partial_json` is valid JSON and equals the concatenated upstream `function.arguments` for that index. Each block carries the id and name first seen for that index.

### 2. Non-contiguous or unmatched content_block indices
- **Trigger:** The stream has thinking, then text, then tool_use. The converter reuses the upstream `tool_calls[].index` (or a counter shared across block types) as the Anthropic block index. tool_use then starts at index 4, and `content_block_stop` fires for indices 2 and 3, which were never opened. The Anthropic SDK raises IndexError. A related CCR case: a second reasoning segment after tool_use gets no new `content_block_start`, and the tool_use block is never stopped.
- **Direction:** stream C->A, stream R->A
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/5483 ; CCR https://github.com/musistudio/claude-code-router/issues/1725
- **Property:** Block indices form 0,1,2,... in order of `content_block_start`. Every `content_block_delta` and `content_block_stop` refers to the single currently open index. Every start has exactly one stop before the next start and before `message_delta`.

### 3. Duplicate lifecycle events (two content_block_stop, two message_start)
- **Trigger:** A tool call is streamed from an OpenAI-compatible upstream. The bridge closes the tool_use block twice, so the client runs the tool twice. In another case, `message_start` is emitted twice when `reasoning_content` is present.
- **Direction:** stream C->A, stream R->A
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/37273 ; LL https://github.com/BerriAI/litellm/issues/32357
- **Property:** Exactly one `message_start` (first event), exactly one `message_delta`, and exactly one `message_stop` (last event). The count of `content_block_stop` events equals the count of `content_block_start` events.

### 4. Spurious empty text block / empty deltas
- **Trigger 1:** The first upstream chunk is a tool call with empty arguments. The adapter "always assumes text first" and opens an empty text block at index 0.
- **Trigger 2:** A chunk has `delta.content` plus `tool_calls: []`. The presence of the key, not its contents, is treated as tool use, so text becomes empty `input_json_delta` events.
- **Direction:** stream C->A
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/31561 ; LL https://github.com/BerriAI/litellm/issues/29286 ; LL https://github.com/BerriAI/litellm/issues/31560
- **Property:** No text block is opened unless a non-empty text delta follows. An empty `tool_calls` array, or `null`, never opens a tool_use block. Every non-empty `delta.content` lands in a `text_delta`.

### 5. Tool-argument tail lost / truncated (invalid JSON, input becomes {})
- **Trigger:** A streaming tool call through an OpenAI-compatible channel loses its last `input_json_delta` fragment(s), typically the closing `"}`. This happens when pending arguments are not flushed at `finish_reason`, or when arguments arrive in the same chunk as `finish_reason`. A reasoning transformer also bumped `choices[0].index` after reasoning ended and copied `tool_calls` into a thinking chunk.
- **Direction:** stream C->A
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7302 ; NA https://github.com/QuantumNous/new-api/issues/4203 ; CCR https://github.com/musistudio/claude-code-router/issues/1397
- **Property:** For every tool call, the concatenation of emitted `partial_json` is byte-equal to the concatenation of upstream argument fragments, including fragments carried in the chunk that also has `finish_reason`. `json.Valid` holds on the result whenever it holds upstream.

### 6. Result depends on how SSE bytes are split across reads
- **Trigger:** The `finish_reason` line and the trailing usage line arrive in one network read. The `finish_reason` branch does `break`, which discards the remaining lines already in the buffer, so usage is reported as 0. The bug is intermittent because it depends on TCP segmentation.
- **Direction:** stream C->A (SSE parser)
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1422
- **Property:** Feeding the same SSE byte sequence split at every possible boundary (byte-by-byte, line-by-line, all-at-once) yields identical output events. A good fuzz or property test.

### 7. Trailing / late usage ignored (first usage wins, or non-empty choices on usage chunk)
- **Trigger 1:** The upstream sends usage on every chunk. The `finish_reason` chunk has `cached_tokens: 0`, and the true value arrives in a final chunk with `choices: []`. The proxy emits `message_delta` at `finish_reason` and never merges the later value.
- **Trigger 2:** The final usage chunk has `choices: [{"index":0,"delta":{}}]` instead of `[]`, so it is not recognized as the usage chunk.
- **Trigger 3:** In general, usage is 0 when `stream_options.include_usage` is not requested.
- **Direction:** stream C->A, usage
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7472 ; LL https://github.com/BerriAI/litellm/issues/36168 ; CCR https://github.com/musistudio/claude-code-router/issues/1187 ; CCR https://github.com/musistudio/claude-code-router/issues/1587
- **Property:** The usage emitted in the final `message_delta` equals the usage from the last upstream chunk that carried usage, whatever its `choices` length. The converter defers `message_delta` until upstream EOF or `[DONE]`. Requests sent upstream with `stream=true` set `stream_options.include_usage=true`.

### 8. Text dropped when a chunk has both content and reasoning_content
- **Trigger:** A DeepSeek or Kimi style upstream sends one chunk with both `delta.reasoning_content` and `delta.content`. The streaming translator returns only the thinking delta, so the final `/v1/messages` result has `content: []`. The non-stream path was fine.
- **Direction:** stream C->A
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/27492
- **Property:** For every chunk, every non-empty field among `content`, `reasoning_content` and `tool_calls` produces output. The multiset of text characters out equals in, and the same holds for reasoning.

### 9. Delta type does not match block type (thinking inside a text block, text inside a tool_use block)
- **Trigger:** `reasoning_content` is streamed as `thinking_delta` inside a block started as `text`, so Claude Code sees empty output. Or a text delta lands on an open tool_use block ("Content block is not a text block").
- **Direction:** stream C->A, stream R->A
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/32357 ; CCR https://github.com/musistudio/claude-code-router/issues/1725 ; CCR https://github.com/musistudio/claude-code-router/issues/1397
- **Property:** `text_delta` only appears in text blocks, `thinking_delta` and `signature_delta` only in thinking blocks, and `input_json_delta` only in tool_use blocks. A thinking block opened from reasoning ends with a `signature_delta`, even an empty one, if the target client requires it.

### 10. Inline `<think>` reasoning / fragmented reasoning not normalized
- **Trigger 1:** The upstream (MiniMax M3 and similar) puts reasoning inline in `content` as `<think>...</think>`, and it is forwarded as visible text.
- **Trigger 2:** A non-stream reply aggregated from a reasoning stream produced 612 one-fragment thinking blocks and no text block.
- **Direction:** response C->A (stream and non-stream)
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/7271 ; CCR https://github.com/musistudio/claude-code-router/issues/1793
- **Property:** A non-stream response has at most one thinking block per contiguous reasoning run, and adjacent same-type fragments are coalesced. If inline-think parsing is supported, no `<think>` tag survives in a text block.

### 11. `[DONE]` matched as substring
- **Trigger:** Tool arguments or text contain the literal `[DONE]`. The iterator does a substring check before JSON parsing and ends the stream early as a normal stop.
- **Direction:** SSE parser (any stream)
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/31562
- **Property:** A stream terminates only when the payload after `data:`, trimmed, is exactly `[DONE]`. A data payload `{"x":"[DONE]"}` is parsed as JSON.

### 12. Truncated stream (EOF without terminal event) reported as success
- **Trigger:** The upstream closes cleanly after a partial delta, for example with unfinished JSON, without `finish_reason`, `message_stop` or `response.completed`. The proxy synthesizes `finish_reason: "stop"` or `end_turn` and logs status 200. The circuit breaker and failover never fire. CCS also hard-closes the client socket instead of sending an SSE error event.
- **Direction:** stream (all directions), routing
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/40260 ; CCS https://github.com/farion1231/cc-switch/issues/6672 ; CCS https://github.com/farion1231/cc-switch/issues/7425
- **Property:** If the upstream stream ends without a terminal marker, the converter emits a protocol-correct error event (Anthropic `event: error`, or R `response.failed`), not `message_stop` with `end_turn`. The attempt is recorded as a failure for health and failover.

### 13. SSE comment heartbeats swallowed
- **Trigger:** The upstream sends `: keep-alive` comments mid-stream. The scanner resets its idle timer but does not forward anything, so a downstream proxy or client idles out on a long reasoning pause.
- **Direction:** stream passthrough / conversion
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/6981
- **Property:** After the first data frame is written, upstream comment or ping lines cause a downstream write and flush (or the converter emits its own `ping`). Comments before the first data frame do not commit a 200.

## B. Stop reason / finish reason mapping

### 14. Incomplete finish_reason <-> stop_reason mapping
- **Trigger 1:** `content_filter` maps to `end_turn`.
- **Trigger 2:** Anthropic `model_context_window_exceeded` maps to `stop` / `completed` instead of `length`.
- **Trigger 3:** With `response_format` plus user tools, a streamed user tool call ends with `stop` instead of `tool_calls`.
- **Trigger 4:** Arguments arrive only in R `response.function_call_arguments.done`, so the finish reason is `stop`.
- **Trigger 5:** The max-tokens limit is reported as `stop`.
- **Direction:** response O->A and A->O (stream and non-stream), R->C
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/40857 ; LL https://github.com/BerriAI/litellm/issues/43012 ; LL https://github.com/BerriAI/litellm/issues/45082 ; LL https://github.com/BerriAI/litellm/issues/25825
- **Property:** The mapping is total over known values: `stop`→`end_turn`, `length`→`max_tokens`, `tool_calls`/`function_call`→`tool_use`, `content_filter`→`refusal`. Anthropic `max_tokens` and `model_context_window_exceeded` map to `length`, `pause_turn` is handled explicitly, and `stop_sequence` maps to `stop`. An unknown value is not silently mapped to success. Invariant: if the output contains at least one tool_use or tool_call, the stop reason is `tool_use` / `tool_calls`. Streaming and non-streaming give the same reason.

## C. Responses <-> Chat / Responses <-> Anthropic

### 15. Function arguments sent only in `.done` event are dropped
- **Trigger:** An R stream sends `response.function_call_arguments.done` (or the final `output_item.done`) with arguments and no preceding `.delta` events. The converter only handles deltas, so the tool call has `arguments: "{}"`.
- **Direction:** stream R->C, R->A
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/27144
- **Property:** For each function_call item, the emitted arguments equal the arguments in the `.done` / `output_item.done` event when no deltas were seen, and equal the concatenated deltas otherwise. Never `{}` when the upstream gave non-empty arguments.

### 16. Required-but-empty fields dropped by `omitempty` (reasoning.summary)
- **Trigger:** A C->R stream converts `reasoning_content` into a reasoning item. The `Summary` field is tagged `omitempty`, so `"summary": []` is omitted, or set to `null` in another implementation. Strict SDKs (openai-php, Codex CLI) then throw.
- **Direction:** stream and non-stream C->R, A->R
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7832 ; LL https://github.com/BerriAI/litellm/issues/40519
- **Property:** The marshalled JSON of every reasoning item contains `"summary"` as an array (never absent, never null). More generally, grep Go structs for `omitempty` on fields the spec marks required: `content: []`, `summary: []`, `input: {}` for tool_use, and `arguments`.

### 17. Empty or missing tool name / id propagated
- **Trigger 1:** An upstream Chat tool call has an empty `function.name`, and it is converted into an R `function_call` with an empty name, which breaks history replay.
- **Trigger 2:** An Anthropic `server_tool_use` arrives without `id`, or `web_search_tool_result` without `tool_use_id`. The converter then hits a nil-pointer panic, and the failed stream is still billed.
- **Direction:** response C->R, A->R
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7094 ; NA https://github.com/QuantumNous/new-api/issues/7549
- **Property:** Every emitted tool call or function_call has a non-empty name and a non-empty id / call_id. A missing upstream id is synthesized deterministically, for example from the block index. The converter never panics on missing optional fields (fuzz with fields deleted).

### 18. call_id / tool_use_id pairing lost across conversions
- **Trigger 1:** Anthropic history (tool_use plus tool_result) is converted to R `input[]`. A `function_call_output` lacks `call_id`, or its call_id does not match a prior `function_call`, or the code confuses `call_id` with the item id `fc_...`. The upstream answers "No tool output found for tool call".
- **Trigger 2:** A Gemini→Chat conversion fails to pair tool_call_id.
- **Direction:** request A->R, A->C
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1643 ; CCR https://github.com/musistudio/claude-code-router/issues/1781 ; NA https://github.com/QuantumNous/new-api/issues/6602
- **Property:** In converted R input, every `function_call_output.call_id` equals the `call_id` of an earlier `function_call` item, and every `function_call` has exactly one output. The same holds for Chat: every `role:"tool"` message's `tool_call_id` matches a `tool_calls[].id` in the immediately preceding assistant message.

### 19. Upstream ids leaked with the wrong prefix
- **Trigger:** An R→A stream copies the upstream `resp_*` id into Anthropic `message_start.message.id`. Older Claude Code versions then render the turn twice.
- **Direction:** stream R->A
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/6758
- **Property:** Anthropic `message.id` starts with `msg_`, OpenAI chat ids start with `chatcmpl-`, and R ids start with `resp_`, whatever the upstream format.

### 20. Usage ignored on non-"completed" terminal events
- **Trigger:** A streaming `/v1/responses` request ends with `response.incomplete` (max_output_tokens hit), `failed` or `cancelled`. Usage is read only in the `response.completed` branch, so the request is billed as 0.
- **Direction:** stream R (usage)
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7241
- **Property:** Usage is extracted from every terminal R event that carries `response.usage`. `incomplete` maps to `max_tokens` / `length`, not to success with zero usage.

### 21. Thinking history mis-converted (to visible text, dropped, or not replayed as reasoning_content)
- **Trigger 1:** An assistant `thinking` block replayed to an R model is serialized as `output_text`, which leaks private reasoning as visible text.
- **Trigger 2:** In A→C, plaintext thinking is not mapped to `reasoning_content` on assistant tool-call turns. DeepSeek and Kimi then return 400 ("reasoning_content is missing in assistant tool call message").
- **Trigger 3:** An empty `reasoning_content` is dropped by a truthiness check.
- **Trigger 4:** Thinking blocks in history sent to an `openai_responses` upstream cause a 400.
- **Direction:** request A->C, A->R
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/26916 ; LL https://github.com/BerriAI/litellm/issues/37270 ; NA https://github.com/QuantumNous/new-api/issues/4795 ; CCR https://github.com/musistudio/claude-code-router/issues/1686
- **Property:** A thinking block in an assistant turn maps to `reasoning_content` (Chat) or a `reasoning` item (R), never to text or `output_text`. On assistant turns that contain tool calls, `reasoning_content` is present (possibly `""`) when the source had a thinking block. `redacted_thinking` and signatures are never turned into text.

### 22. Thinking signatures invalid after provider switch / failover
- **Trigger:** A session is started on provider X with signed thinking blocks, then fails over (429) or is switched to provider Y. Y rejects the foreign signatures, or requires thinking to be passed back ("content[].thinking in the thinking mode must be passed back"). Stripping the blocks without disabling thinking leaves every later request failing with 400.
- **Direction:** request A->A (cross-provider), routing
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/7226 ; CCS https://github.com/farion1231/cc-switch/issues/2401 ; CCS https://github.com/farion1231/cc-switch/issues/3930
- **Property:** When the target provider differs from the provider that produced a thinking block (signature origin), the request is consistent. Either signatures are kept only for the same provider, or thinking blocks are stripped and thinking is not left enabled with tool-use turns lacking thinking. Stripping is idempotent and does not touch same-provider passthrough (see CCS #3300).

## D. Request conversion A -> C (and C -> A)

### 23. Assistant text dropped when tool calls are present
- **Trigger:** An Anthropic assistant turn has `[text, tool_use]`. The converter sets content only when `len(toolCalls)==0`, so the text is lost and `content: null`.
- **Direction:** request A->C
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/6615
- **Property:** Every text block in an assistant turn appears in the output assistant message's `content` alongside `tool_calls`.

### 24. User text next to tool_result dropped; tool_result dropped on rewrite
- **Trigger 1:** A user turn has `[tool_result..., text]`. The tool_results become `role:tool` messages and the trailing text is silently lost.
- **Trigger 2:** A content rewrite (the `/compact` handoff) replaces the whole content array, dropping a `tool_result` so a tool_use goes unanswered.
- **Direction:** request A->C
- **Evidence:** FCP https://github.com/fuergaosi233/claude-code-proxy/issues/86 ; CCR https://github.com/musistudio/claude-code-router/issues/1823
- **Property:** Conservation: the count of text, image and tool_result blocks in equals the count of corresponding parts or messages out. Every Anthropic tool_use id has exactly one matching tool message.

### 25. Message ordering violates strict upstream rules (tool messages not contiguous)
- **Trigger:** Parallel tool results include images. The converter moves the images into a separate user message and flushes it between two `role:tool` messages, which DeepSeek rejects with 400.
- **Direction:** request R->C, A->C
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/8521
- **Property:** In Chat output, all `role:tool` messages answering an assistant turn come immediately after it with no interleaved messages. Any extra user content, such as images, is placed after the last tool message.

### 26. tool_choice semantics lost
- **Trigger:** `tool_choice: {"type":"any"}` is translated to `"auto"`, so the forced tool call is lost. Related: `disable_parallel_tool_use` / `parallel_tool_calls:false` and `strict` are dropped.
- **Direction:** request A->C (and C->A)
- **Evidence:** FCP https://github.com/fuergaosi233/claude-code-proxy/issues/87 ; LL https://github.com/BerriAI/litellm/issues/41913
- **Property:** `any`→`"required"`, `auto`→`"auto"`, `none`→`"none"`, and `{type:tool,name}`→`{type:function,function:{name}}`, with the reverse for C->A. `disable_parallel_tool_use:true` ⇔ `parallel_tool_calls:false`.

### 27. Tool definitions silently dropped (nil parameters / schema)
- **Trigger:** A Chat tool with no `parameters` fails a type assertion `.(map[string]any)` and is skipped without an error. An unchecked `params["type"].(string)` can panic.
- **Direction:** request C->A (Go)
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/6859
- **Property:** `len(out.Tools) == len(in.Tools)`. A nil or absent schema becomes `{"type":"object","properties":{}}`. No unchecked type assertions on user JSON (fuzz).

### 28. Images dropped (top-level or inside tool_result) / media_type wrong
- **Trigger 1:** An Anthropic image block is dropped going to Chat, and the request still returns 200.
- **Trigger 2:** An `image_url` inside a Chat tool message is not converted to an Anthropic image inside `tool_result`.
- **Trigger 3:** An image inside a tool_result is sent as base64 text.
- **Trigger 4:** `media_type` is taken from the data-URI prefix (`image/png`) while the bytes are JPEG, and Anthropic returns 400.
- **Direction:** request A->C, C->A
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1678 ; NA https://github.com/QuantumNous/new-api/issues/6260 ; LL https://github.com/BerriAI/litellm/issues/23712 ; LL https://github.com/BerriAI/litellm/issues/24900
- **Property:** The image count is preserved per message, including inside tool_result. Data-URI parsing round-trips (`data:<mt>;base64,<d>` ⇔ `{type:base64, media_type:mt, data:d}`). Optionally, media_type matches `http.DetectContentType(bytes)`.

### 29. Per-block flags and params dropped (is_error, stop_sequences, developer role, cache_control)
- **Trigger 1:** `tool_result.is_error:true` becomes a plain tool message.
- **Trigger 2:** `stop_sequences` is not forwarded, so the model emits the stop token and returns `end_turn`.
- **Trigger 3:** The `developer` role is rewritten to `system`.
- **Trigger 4:** `cache_control` is dropped on the A→R bridge.
- **Direction:** request A->C, A->R
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/44979 ; LL https://github.com/BerriAI/litellm/issues/37118 ; LL https://github.com/BerriAI/litellm/issues/41424
- **Property:** A field-coverage table test: for each documented request field, either it maps to a target field or the converter explicitly rejects or logs it. `stop_sequences` ⇔ `stop`. An `is_error` tool result is marked in output (prefix or field).

### 30. Empty text blocks forwarded to strict upstream
- **Trigger:** A prior streamed assistant turn began with an empty text block, which Claude Code replays. Bedrock and other strict upstreams return 400 "text field is blank".
- **Direction:** request A->X
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/26554
- **Property:** The output never contains a text part with `""` content, and never a message with an empty content array (unless the message has tool_calls).

### 31. Numeric parameter mapping saturates or violates bounds (budget_tokens → effort, max_tokens floor)
- **Trigger 1:** `thinking.budget_tokens` 16000, 32000 or 64000 all map to `reasoning_effort: "high"` (the ladder caps out).
- **Trigger 2:** `max_tokens: 1` (Claude Code title/classifier calls) is forwarded below the upstream minimum (16), giving a 400.
- **Trigger 3:** `max_tokens` vs `max_completion_tokens` confusion for reasoning models.
- **Direction:** request A->C, A->R
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7508 ; CCR https://github.com/musistudio/claude-code-router/issues/1781 ; LL https://github.com/BerriAI/litellm/issues/23836
- **Property:** The budget→effort mapping is monotone and its ranges cover every supported effort level. `max_tokens` is clamped to the provider's [min,max]. Reasoning models get `max_completion_tokens`.

### 32. Tool-call id format not sanitized, or sanitized inconsistently
- **Trigger 1:** An OpenAI-protocol upstream returns ids like `Bash:0`, which are forwarded into Anthropic `tool_use.id`. Later Anthropic requests fail with a pattern error `^[a-zA-Z0-9_-]+$`.
- **Trigger 2:** An id sanitizer rewrites tool_use ids but not the matching ids in passthrough or replayed history (Kimi `functions.x:0`), which breaks multi-turn pairing.
- **Direction:** response C->A (ids), request A->C (history)
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1808 ; LL https://github.com/BerriAI/litellm/issues/32214 ; LL https://github.com/BerriAI/litellm/issues/34522
- **Property:** Every id emitted in Anthropic format matches `^[a-zA-Z0-9_-]+$`. The sanitize function is deterministic and injective on a conversation and is applied identically to tool_use.id and tool_result.tool_use_id, so pairing is preserved.

### 33. System prompt handling breaks caching or semantics
- **Trigger 1:** Claude Code's per-request `x-anthropic-billing-header: cch=<hash>` line at the top of the system prompt is forwarded upstream, so the prompt prefix changes every turn and the upstream cache never hits.
- **Trigger 2:** Mid-conversation system reminders are folded into R `instructions`, so the prefix is unstable.
- **Trigger 3:** A `messages[].role=system` entry is not moved to top-level `system` for Anthropic backends.
- **Direction:** request A->C, A->R, C->A
- **Evidence:** RCP https://github.com/1rgs/claude-code-proxy/issues/99 ; LL https://github.com/BerriAI/litellm/issues/40198 ; CCS https://github.com/farion1231/cc-switch/issues/3750
- **Property:** For two requests that differ only in the billing-header line, the converted upstream bodies have identical system prefixes. In C->A, all system and developer messages end up in top-level `system` in order, and none remain in `messages`.

## E. Usage / token accounting

### 34. Cached-token basis mismatch (input_tokens includes or excludes cache; cache fields dropped)
- **Trigger 1:** OpenAI `prompt_tokens` includes `cached_tokens`, but O→A conversion copies `prompt_tokens` into Anthropic `input_tokens` without subtracting. Clients sum input + cache_read and double-count, and Claude Code compacts at half the real context.
- **Trigger 2:** `cache_creation_input_tokens` is dropped (always 0).
- **Trigger 3:** `cache_read_input_tokens` is 0 on bridges.
- **Trigger 4:** `cached_tokens` is greater than `input_tokens` because two bases are mixed.
- **Direction:** usage, response O->A / A->O / R
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1655 ; CCR https://github.com/musistudio/claude-code-router/issues/1461 ; LL https://github.com/BerriAI/litellm/issues/31069 ; LL https://github.com/BerriAI/litellm/issues/40277
- **Property:** O→A: `input_tokens = prompt_tokens - cached_tokens - cache_creation` (≥0), `cache_read_input_tokens = prompt_tokens_details.cached_tokens`. A→O: `prompt_tokens = input + cache_read + cache_creation` and `cached_tokens = cache_read`. Round-trip is the identity, and `cached_tokens <= prompt_tokens` always.

### 35. Double counting / wrong splits in cost and usage records
- **Trigger 1:** Cache-read tokens are billed at both the input rate and the cache rate.
- **Trigger 2:** A streaming cache write is split into 5m and 1h buckets merged independently from `message_start` and `message_delta`, so 5m + 1h is greater than the total.
- **Trigger 3:** The same request is logged twice (proxy row plus session import with a different dedupe key).
- **Direction:** usage / billing / stats
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/40006 ; NA https://github.com/QuantumNous/new-api/issues/7777 ; CCS https://github.com/farion1231/cc-switch/issues/8415
- **Property:** `ephemeral_5m + ephemeral_1h == cache_creation_input_tokens`. Cost = input×p_in + cache_read×p_cr + cache_write×p_cw + output×p_out, with each token counted once. The usage store is idempotent per request id.

### 36. Explicit zero vs missing usage; start-of-stream zeros
- **Trigger 1:** A stream ends with explicit `input_tokens: 0, output_tokens: 0` (refusal) plus cache usage. Finalization treats 0 as "missing" and substitutes a local estimate, which double-charges.
- **Trigger 2:** `message_start.usage.input_tokens` is 0 on the R→A bridge because R reports usage only at the end.
- **Trigger 3:** A local estimator overestimates 5x on `\uXXXX`-escaped JSON.
- **Direction:** usage (stream)
- **Evidence:** NA https://github.com/QuantumNous/new-api/issues/7815 ; LL https://github.com/BerriAI/litellm/issues/41406 ; NA https://github.com/QuantumNous/new-api/issues/7368
- **Property:** Usage fields use pointers or "present" flags, so an upstream-reported 0 is never replaced by an estimate. Estimation operates on decoded strings, not raw JSON bytes.

## F. Errors

### 37. Error not translated to the inbound protocol / status mapping
- **Trigger 1:** An upstream 503 (Bedrock capacity) on an Anthropic `/v1/messages` endpoint is returned as an OpenAI-shaped error body, so Claude Code does not recognize `overloaded_error` and does not retry.
- **Trigger 2:** A quota error is swallowed and retried endlessly instead of being surfaced.
- **Trigger 3:** HTTP 200 with a body of `null` is returned after repeated upstream 422s.
- **Direction:** errors, response O->A / A->O
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/36655 ; RCP https://github.com/1rgs/claude-code-proxy/issues/89 ; LL https://github.com/BerriAI/litellm/issues/32221
- **Property:** The error body schema matches the inbound API (A: `{"type":"error","error":{"type",...}}`; C/R: `{"error":{"message","type","code"}}`). Status mapping: 429→`rate_limit_error`, 503/529→`overloaded_error`, 401→`authentication_error`, 400→`invalid_request_error`. The upstream message is preserved. A 2xx is never returned with an empty or null body.

## G. Routing / failover

### 38. Fallback attempt hits the same provider or keeps the old model id
- **Trigger 1:** Model-chain fallback logs attempts to other providers, but per-attempt headers still point to the primary, so every retry goes to the dead primary.
- **Trigger 2:** Failover to a native-Responses backup keeps the previous provider's model id, which the backup rejects.
- **Trigger 3:** After a successful fallback, the response `model` field shows the primary model.
- **Direction:** routing
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1804 ; CCS https://github.com/farion1231/cc-switch/issues/7547 ; CCR https://github.com/musistudio/claude-code-router/issues/1861
- **Property:** For attempt i, the outbound URL, credential and model equal provider_i's configuration (base_url, key, mapped model), for every protocol path. The model reported in the response or logs equals the model that served it.

### 39. Wrong retry/failover classification; retry index skipping
- **Trigger 1:** An upstream 400 "Model is unavailable" is classified NonRetryable, so there is no failover.
- **Trigger 2:** A `response.completed` with no output is accepted as success, which causes retry storms.
- **Trigger 3:** Channel affinity blocks failover on a retryable 429.
- **Trigger 4:** After the top channel is auto-disabled and removed from the cached list, the cumulative retry index skips the next-best channel.
- **Trigger 5:** A client-cancelled stream is counted as a model failure, and the upstream is not cancelled on client disconnect.
- **Direction:** routing / health
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/6821 ; CCS https://github.com/farion1231/cc-switch/issues/8228 ; NA https://github.com/QuantumNous/new-api/issues/6095 ; NA https://github.com/QuantumNous/new-api/issues/4184
- **Property:** The selection on retry is "highest-priority provider not yet tried in this request" (a tried-set, not an index into a mutable list). Client-cancel (`ctx.Err()==context.Canceled`) is neither a provider failure nor retried, and the upstream request uses the client's context. The retryable-status table is explicit and tested (429, 5xx, timeouts, and selected 400 bodies).

### 40. Circuit-breaker half-open permit leak
- **Trigger:** A half-open probe exits abnormally (failover continue, select cancellation, early return, panic) without releasing its permit. With `max_half_open_requests=1`, the provider stays stuck in HalfOpen.
- **Direction:** routing / health
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/7206
- **Property:** In Go, the permit is released with `defer`. After any sequence of acquire→(success|failure|cancel|panic), the in-flight count returns to 0.

## H. Credentials / config / admin

### 41. Credential leak via URLs, errors, logs, telemetry
- **Trigger 1:** A Gemini key in the `?key=` query string appears in exception text and logs on 404.
- **Trigger 2:** Retry metadata (`previous_models`) copies all kwargs, including the API key and prompts, into OTel spans.
- **Direction:** credentials
- **Evidence:** LL https://github.com/BerriAI/litellm/issues/15799 ; LL https://github.com/BerriAI/litellm/issues/13705 ; LL https://github.com/BerriAI/litellm/issues/8075
- **Property:** For any upstream failure, the error string returned to the client and every log line contain no configured secret. Test by setting a sentinel key and grepping all outputs. `*url.Error` messages are redacted.

### 42. Admin/config endpoint exposes or rewrites secrets
- **Trigger 1:** An unauthenticated `GET /` redirects to a URL containing the management token (Docker nginx), and forged `Origin`/`Referer` headers bypass the origin check.
- **Trigger 2:** The UI does GET `/api/config` (env vars expanded), then POST `/api/config`, which persists literal keys in place of `${VAR}` references.
- **Direction:** credentials / config / admin
- **Evidence:** CCR https://github.com/musistudio/claude-code-router/issues/1835 ; CCR https://github.com/musistudio/claude-code-router/issues/1373
- **Property:** The admin GET masks secrets, and a POST of the masked value does not overwrite the stored secret. Env-var references round-trip unchanged through load→save. Admin endpoints require auth independent of Origin/Referer.

### 43. Config write clobbers unmanaged keys or corrupts the file
- **Trigger 1:** Applying a partial provider config overwrote `~/.claude/settings.json`, dropping 22 of 23 keys.
- **Trigger 2:** The writer regenerated `config.toml` from a snapshot, deleting unmanaged `[model_providers.*]` tables.
- **Trigger 3:** An in-place, non-truncating write of shorter JSON left trailing bytes, the next load failed, and settings reset to defaults.
- **Trigger 4:** DB row and live file got out of sync after a failed write.
- **Direction:** config
- **Evidence:** CCS https://github.com/farion1231/cc-switch/issues/6902 ; CCS https://github.com/farion1231/cc-switch/issues/6860 ; CCS https://github.com/farion1231/cc-switch/issues/8429 ; CCS https://github.com/farion1231/cc-switch/issues/7791
- **Property:** load→modify(one field)→save preserves all unknown keys byte-for-byte or semantically. Saves are atomic (temp file plus rename, or `O_TRUNC`). A failed save leaves both the in-memory and on-disk state unchanged. A parse error on load does not silently replace the config with defaults.

---

## Most transferable checks for a Go gateway (quick list)
1. SSE parser chunk-boundary invariance (#6) and exact `[DONE]` match (#11).
2. Anthropic event grammar validator: one message_start, contiguous indices, start/stop pairing, delta-type equals block-type (#2, #3, #9).
3. Per-tool-index argument conservation, including arguments in the finish chunk and in R `.done` (#1, #5, #15).
4. Content conservation in request conversion: text, image and tool_result counts, plus pairing of tool ids (#18, #23, #24, #28).
5. Total stop-reason maps, plus "tool_use present ⇒ stop_reason tool_use" (#14).
6. Usage basis conversion round-trip, cached ≤ prompt, last-usage-wins in streams, nil-vs-0 (#7, #34, #36).
7. `omitempty` audit on required fields (#16).
8. Retry selection via a tried-set, client-cancel handling, and correct per-attempt model and credential (#38, #39).
9. Secret-sentinel grep over errors and logs, and admin GET/POST masking round-trip (#41, #42).
10. Config save atomicity and unknown-key preservation (#43).
