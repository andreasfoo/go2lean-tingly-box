# Triage of the seventh pass

The 162 confirmed findings of the seventh pass that are not nil inputs
(106 + 56 nil dereferences with no nil input), one row each in
`seventh-triage.tsv` (category, package, function, panic).

| category | count | what it is |
|---|---|---|
| test file | 31 | helpers declared in `_test.go` files (fakes, scripted models, test body builders); go2lean no longer targets them (`TestFile`) |
| test double | 10 | the `protocoltest` package and fake/mock/scripted streams, decoders and forwarders |
| truncate helper | 6 | `truncate(s, n)`, `shortenPath`, `snippetHead`, `sliceLines` with a negative `n`; every caller passes a constant |
| to check | 115 | the rest, read below |

The 115, by kind:

- **Dependencies not wired** (about 60): `internal/command` `*CmdKong.Run`
  and the `run*` helpers dereference the app or its services, nil in a
  receiver the test built field by field; `[db] Provider store missing`
  (17) is an explicit panic when the store was never set. Production
  code wires them at start-up.
- **Typed dispatch on the wrong request** (13): `ProtocolHandler.dispatch*`
  and friends assert the parsed request to the type their route parsed;
  a nil `any` request never comes from the router.
- **Documented preconditions** (about 10): `err is nil` panics in
  `SendForwardingError`/`SendStreamingError`/`failForward` (callers pass
  a non-nil error), `SelectPendingRequest` ("pendingReqs must be
  non-empty", its only caller returns early on empty), `MetadataUserID.Fix`
  ("missing device id"), `MustLoadPrompt`.
- **Guarded by every caller** (checked one by one): `mostSpecificTemplate`
  (called only with more than one candidate), `blobPath` (callers check a
  64-character hex hash), `TierTactic.pickWithinTier` (both call sites
  pass a non-empty slice), `buildWanxEditBody`/`editWanx`
  (`EditRequestFromOpenAI` rejects a request without an image),
  `renderProgressBar` (`Countable` guarantees a positive limit; the only
  caller passes width 20), `ServiceStats.Record*` (the only caller passes
  100 samples), `detectShellOperator` (index from the scanning loop).
- **Receiver state set by the test** (iterators' `current`/`next` with an
  index field of any value, `MemoryLogHook.Fire` with a zero capacity).

No finding of this group is reported. The conclusion for go2lean: past
nil inputs, a templated `no_panic` on every function mostly finds inputs
the callers rule out; the next filter is to require a caller to reach
the same panic (a finding in a function every caller guards is not
reported), and the spec-driven rounds remain where the bugs come from.
