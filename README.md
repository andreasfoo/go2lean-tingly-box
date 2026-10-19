# go2lean on tingly-box

What [go2lean](https://github.com/andreasfoo/go2lean) found when pointed at
[tingly-box](https://github.com/tingly-dev/tingly-box), an LLM gateway
that converts requests and responses between the Anthropic, OpenAI Chat
and OpenAI Responses protocols, with and without streaming. go2lean
translates Go to a Lean model, searches the model for inputs that refute
a property, and confirms each candidate with `go test` on the real code.

The tool lives in its own repository; this one holds what is specific to
tingly-box: the properties, the runs, the evidence, the findings.

## Layout

| path | contents |
|---|---|
| `findings/` | one page per finding: what happens, where in the source, the spec that found it, its status upstream |
| `specs/` | hand-written properties (spec-in-Go) per tingly-box package, as `_test.go` files |
| `guidance/` | guidance given to go2lean's LLM agent when it writes properties |
| `agent-runs/` | what the agent proposed and reported |
| `poc/` | proof-of-concept tests for confirmed counterexamples, per round |
| `passes/` | full passes over the whole tree, with their counts |
| `hunting.md` | the rounds in order: properties, engine output, verdicts, model changes |
| `scripts/` | install the specs, run a round on chosen properties, run a resumable full pass |

## Running

```sh
# a tingly-box checkout to test, and a go2lean checkout
scripts/install-specs.sh ~/src/tingly-box
# hunt the properties whose id matches a pattern, verify with go test
scripts/round.sh ~/src/go2lean ~/src/tingly-box /tmp/ws 'roundtrip' /tmp/round.log
# every package (hours; rerun to resume after an interruption)
scripts/full-pass.sh ~/src/go2lean ~/src/tingly-box out-pass /tmp/pass.log
```

go2lean needs Go and Lean 4 (elan); see its README.

## Status

Findings marked **new** have not been reported upstream yet.
