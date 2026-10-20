# Finding 36: budget-thinking Claude models: thinking blocks stripped from the tool turn while thinking stays on (new, not reported)

Severity: **high** (see [the findings table](README.md)).

Path: Anthropic client (V1 and Beta) -> Anthropic provider at
`api.anthropic.com` / `claude.ai` or a Claude Code OAuth provider (the vendor
transform's Claude Code backend branch), for a model with budget thinking
and no adaptive thinking: Sonnet 4.5, Haiku 4.5, Opus 4.5, Opus 4.1, Opus 4,
Sonnet 4, Sonnet 3.7 (`internal/catalog/claude.models.json`).

Where (tingly-box `b788a3b`), `internal/protocol/ops/request_anthropic_model.go`:

- `:96-97` (`ApplyAnthropicV1ModelTransform`) and `:141-142`
  (`ApplyAnthropicBetaModelTransform`): for every model without adaptive
  thinking, all messages go through the thinking filter, whatever the
  thinking config; the config is changed only when it is adaptive
  (`:98-108`, `:143-153`), so `enabled` stays enabled.
- `filterThinkingBlocksInMessages` (`:181`, test `:192`, skip `:208`) and
  `filterBetaThinkingBlocksInMessages` (`:227`, test `:238`, skip `:254`)
  drop every `thinking` block of every message, including the last
  assistant message, the tool-use turn the request continues. A message
  left empty is dropped (`:215`, `:261`). `redacted_thinking` blocks are
  kept, so the filter is not even consistent with itself.
- The vendor transform runs `thinking.ReconcileBetaWithRequest` /
  `ReconcileV1WithRequest` first (`internal/protocol/transform/vendor.go:98-99`,
  `:122-123`). Its `DisableV1ForUnsignedToolTurn`
  (`internal/protocol/thinking/reconcile.go:68-80`) states the rule this
  breaks: "With budget thinking enabled, Anthropic requires that final
  assistant message to start with a (signed) thinking block … Run it before
  any model-specific thinking-block filtering". It sees the client's
  thinking block and keeps thinking on; the model transform then removes the
  block.

Trigger: a request to a budget-thinking model with
`thinking: {type: enabled, budget_tokens: N}` that continues a tool-use
turn: the last assistant message is `[thinking (signed), tool_use]`,
followed by the user's `tool_result`.

Observed: thinking stays `enabled` (budget 10000 for Sonnet 4.5, 4096 for
Haiku 4.5) and the assistant message becomes `[tool_use]`. Anthropic rejects
such a request with 400 ("When `thinking` is enabled, a final `assistant`
message must start with a thinking block").

Expected: the signed thinking block of the final assistant tool-use turn is
kept first (dropping thinking from earlier turns is allowed), or thinking is
disabled for this request.

Realism: high. Claude Code with thinking on and one of these models (Sonnet
4.5 and Haiku 4.5 are common defaults) sends this shape on every step of a
tool loop after the first, so every agentic session that uses a tool breaks
on its second request. Any Anthropic SDK client with extended thinking and
tools hits it the same way.

Status: **new** (tracker search 2026-10-20 over the full text of all issues
and PRs for `budget`, `thinking` with `strip`, `signature`: no issue about
this). Related, not the same: #2004 (open) and its PR #2005 (open) on the
thinking guard (`Reconcile*ThinkingWithRequest`) flipping `thinking` between
requests for third-party providers; this finding is the model transform's
filter that runs after the guard on Anthropic's own backend, which #2005
does not change. Reproduces on main `bc460f6`.

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_BetaBudgetThinkingKeepsToolTurnThinking` and
`TestGo2LeanRepro_V1BudgetThinkingKeepsToolTurnThinking`
(`repro/internal/protocol/ops/zz_go2lean_repro_hunt4_test.go`).

Fix direction: never filter the final assistant turn (the API accepts the
thinking blocks of earlier turns either way), or filter only when thinking
ends up disabled, and treat `redacted_thinking` like `thinking`.

## General property

For every Anthropic request (V1 or Beta) and every model, if the request's
final assistant message starts with a thinking (or redacted_thinking) block,
then after the model transform either thinking is not `enabled`, or the
final assistant message still starts with that same block.

Target functions: `ops.ApplyAnthropicV1ModelTransform` and
`ops.ApplyAnthropicBetaModelTransform`.
