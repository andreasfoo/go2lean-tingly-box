# go2lean on tingly-box

What [go2lean](https://github.com/andreasfoo/go2lean) found in
[tingly-box](https://github.com/tingly-dev/tingly-box), an LLM gateway that
converts requests and responses between the Anthropic, OpenAI Chat and
OpenAI Responses protocols, with and without streaming.

go2lean translates Go into a Lean model, searches the model for inputs
that break a property (written in Go, as a `Spec…` function), and turns
each candidate into a `go test` on the real code: only what the real run
confirms is a finding. This repository holds the tingly-box side: the
properties, the findings with a plain Go reproduction each, and the
record of every run.

## Findings

| | finding | where | status |
|---|---|---|---|
| 1 | [Tool-call ids truncated on one side only](findings/issue-1924-tool-call-ids.md): an Anthropic `tool_use` id over 40 characters reaches the Chat assistant's `tool_calls[].id` untruncated, while the tool message's `tool_call_id` is cut to 40, so the call and its result no longer pair | Anthropic → Chat request | [#1924](https://github.com/tingly-dev/tingly-box/issues/1924) |
| 2 | [System-role messages dropped](findings/issue-1921-system-messages.md): a message with role `system` inside `messages` disappears from the converted request | Anthropic → Chat and Anthropic → Responses requests | [#1921](https://github.com/tingly-dev/tingly-box/issues/1921) |
| 3 | [Parallel tool calls share index 0; `max_tokens` is not `length`](findings/issue-1923-stream-indexes.md): a Chat client merges the streamed calls, and a truncated answer ends with `finish_reason: "stop"` | Anthropic → Chat stream | [#1923](https://github.com/tingly-dev/tingly-box/issues/1923) |
| 4 | [`reasoning_content` empty without streaming](findings/reasoning-content-nonstream.md): the non-streaming converter reads `block.Text` of a `thinking` block (always empty) instead of `block.Thinking`; the streaming converter returns the thinking | Anthropic → Chat, non-streaming | new, not reported |
| 5 | [Refusal lost when streaming](findings/responses-chat-stream-refusal.md): a Responses answer that refuses reaches a non-streaming Chat client as `message.refusal` and a streaming one as an empty answer; the Chat stream chunk type has no refusal field | Responses → Chat stream | new, not reported |

| 6 | [Gateway environment credentials sent to every upstream](findings/env-credentials-upstream.md): the SDK clients keep their environment defaults, so `ANTHROPIC_AUTH_TOKEN` (Claude Code's gateway token) goes out as `Authorization: Bearer` next to the provider's key, and `OPENAI_ORG_ID` / `OPENAI_PROJECT_ID` to every OpenAI-compatible vendor | upstream clients | new, not reported |
| 7 | [Cache reads counted twice without streaming](findings/usage-nonstream-cached-input.md): the non-streaming Chat passthrough records `prompt_tokens` as input next to `cached_tokens`; the streaming path subtracts them (1000/900 records input 1000 vs 100) | usage tracking | new, not reported |
| 8 | [Recordings keep client credentials](findings/recording-client-credentials.md): recorded request headers and URL are not redacted | recording | new, minor |

Findings 1-5 come from go2lean searches. 6-8 come from reading the code
for the next spec directions (usage, credentials, routing, config) and
are confirmed by `go test` (6, 7) or by reading (8). 1-5 reproduce on
tingly-box `5f83f04` (where they were found) and on `b788a3b` (main,
2026-10-20).

## Reproduce in one command

Needs Go and git, nothing from go2lean:

```sh
git clone https://github.com/andreasfoo/go2lean-tingly-box
cd go2lean-tingly-box
scripts/reproduce.sh                       # clones tingly-box main into a temp dir
scripts/reproduce.sh ~/src/tingly-box      # or use a checkout (left as it was)
scripts/reproduce.sh ~/src/tingly-box v1.2.3   # or check out a ref first
```

It copies the tests of `repro/` (one small `go test` per finding, written
as the behaviour a client expects: a failing test is a finding that still
reproduces) into the checkout, runs them, removes them, and prints:

```
test                                  result       finding
ToolCallIDsPairAfterConversion        reproduces   issue-1924-tool-call-ids (tracked #1924)
...
StreamedRefusalReachesChatClient      reproduces   responses-chat-stream-refusal (new)
```

`fixed` means the test passes on that tingly-box. The tests can be taken
into tingly-box as regression tests as they are.

## Running go2lean

go2lean needs Go 1.26 and Lean 4 (installed by `elan`, which picks the
version pinned in go2lean's `lean/lean-toolchain`):

```sh
curl https://raw.githubusercontent.com/leanprover/elan/master/elan-init.sh -sSf | sh
git clone https://github.com/andreasfoo/go2lean
git clone --recurse-submodules https://github.com/tingly-dev/tingly-box
git clone https://github.com/andreasfoo/go2lean-tingly-box
```

The first run builds the Lean side (the engines and one Lean module per
compiled Go package, a few minutes; later runs rebuild only what changed).
Everything below runs from this repository; go2lean writes a workspace of
models, properties, counterexamples and verdicts.

**Check the properties** of the protocol packages (about 45 minutes):

```sh
scripts/update.sh ../go2lean ../tingly-box main /tmp/tb-main
```

It checks the ref out, installs `specs/` into the checkout as
`go2lean_specs_test.go` files, makes sure they still compile, hunts every
spec, verifies each counterexample with `go test`, and writes
`/tmp/tb-main/report.md`: which known finding still reproduces, and every
spec's verdict. Run it whenever tingly-box moves; a spec that no longer
compiles stops it with `go vet`'s message.

**Hunt a few properties** (by a pattern over property ids):

```sh
scripts/install-specs.sh ../tingly-box
scripts/round.sh ../go2lean ../tingly-box /tmp/ws 'roundtrip' /tmp/round.log
```

`BUDGET` (seconds per property, default 120) and `JOBS` (default 2) tune it.

**A full pass** over every package and function (`no_panic` everywhere plus
the installed specs; several hours, resumable):

```sh
scripts/full-pass.sh ../go2lean ../tingly-box out-pass /tmp/pass.log   # rerun to resume
go run ../go2lean/cmd/go2lean summary -workspace ../go2lean/out-pass
```

`passes/` has the result of every pass so far and a triage of the last.

**Write properties with the agent** (an LLM proposes specs, go2lean checks
them):

```sh
cd ../go2lean
go run ./cmd/go2lean agent -deps -guidance ../go2lean-tingly-box/guidance/guidance.md \
  -module ../tingly-box ./internal/protocol/request
```

`ANTHROPIC_API_KEY` must be set; `-base-url` and `-model` point it at any
Anthropic-compatible endpoint.

## Layout

| path | contents |
|---|---|
| `findings/` | one page per finding; `index.tsv` maps each to the specs that show it |
| `repro/` | one plain `go test` per finding, used by `scripts/reproduce.sh` |
| `specs/` | the properties (spec-in-Go), per tingly-box package |
| `guidance/`, `agent-runs/` | guidance given to go2lean's agent, and what it wrote |
| `poc/` | the tests go2lean generated for the confirmed counterexamples, per round |
| `passes/` | full passes: counts per pass, and the triage of the last |
| `hunting.md` | the rounds in order: properties, engine output, verdicts, model changes |
| `scripts/` | `reproduce.sh`, `update.sh`, `round.sh`, `full-pass.sh`, `install-specs.sh` |
