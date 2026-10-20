# The Lean model of tingly-box

What go2lean made of tingly-box: one Lean module per Go package, in
go2lean's GoSSA dialect (Go's SSA form as Lean data: blocks, instructions,
terminators; the dialect, its semantics and the search engines are in
go2lean's [`lean/Go2Lean/`](https://github.com/andreasfoo/go2lean/tree/main/lean/Go2Lean)).

| | |
|---|---|
| tingly-box | `b788a3b` (main of 2026-10-20), with the specs of [`specs/`](../specs) installed |
| go2lean | `0a2eeab` (`compile -deps` of `./ai`, `./internal/protocol` and its subpackages request, stream, nonstream, sse, usage, wire, ops, assembler, as `scripts/round.sh` runs it) |
| [`lean/`](lean) | the generated modules: tingly-box's packages (`github_com_tingly_dev_tingly_box_*`) and the dependencies they call into (the Anthropic, OpenAI and Gemini SDKs, gin, gjson, logrus, ...), as far as the call trees reach, plus `Registry.lean` |
| [`reports/`](reports) | the model report of each tingly-box package: every function with its status (complete / partial / rejected), the reason for partial and rejected ones, and every havoc site; the coverage table in the [README](../README.md#coverage-what-go2lean-translated-into-lean) is made from them by `scripts/coverage.py` |

A function looks like this (`mapOpenAIFinishReasonToAnthropic` in
`internal/protocol/stream`; strings are byte lists):

```lean
/-- func(finishReason string) string (status: complete) -/
def f159 : GoSSA.Func := {
  name := "mapOpenAIFinishReasonToAnthropic", params := [0], nvals := 14, entry := 0,
  blocks := #[
    -- 0: entry
    { params := [],
      instrs := #[.op 1 (.strConst [115, 116, 111, 112]) [],      -- "stop"
                 .op 2 .eq [0, 1]],
      term := .br 2 1 [] 3 [] },
    -- 1: switch.body
    { params := [],
      instrs := #[.op 3 (.strConst [101, 110, 100, 95, 116, 117, 114, 110]) []],  -- "end_turn"
      term := .ret 3 },
    ...
```

Each module also holds the package's JSON table (how its types encode and
decode) and its havoc sites (the calls and values the model does not know;
a search reads them as free inputs, and every counterexample that read one
is still checked by `go test`).

To build or search them, use go2lean itself: `go2lean compile` writes these
modules into `lean/Go2Lean/Generated/gossa/` of a go2lean checkout and
builds them with `lake`; copying these files there and running `lake build`
in go2lean's `lean/` builds the same model (with the go2lean commit above).
They are a snapshot for reading: a later go2lean or tingly-box gives a
different model.
