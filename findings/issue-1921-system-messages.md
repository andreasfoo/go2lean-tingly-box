# Finding 2: System messages dropped (tracked: #1921)

A message with role `"system"` inside `messages` (the request's top-level
`system` empty) is dropped:

- Chat path: the message loop at
  `internal/protocol/request/anthropic_view.go:287-296` handles the user
  and assistant roles only;
- Responses path: `anthropic_v1_to_responses.go:98-106` appends items for
  `"user"` and `"assistant"` only (the real result was an empty input
  list).

Specs: `...KeepsSystemMessages` (both paths). Confirmed; PoCs in
`poc/round2/`.
