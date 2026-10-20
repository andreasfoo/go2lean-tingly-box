# Upstream errors are not translated to the client's protocol (new, not reported)

Path: every converted path whose upstream answers with an HTTP error before
streaming (non-streaming requests, and streaming requests that fail before
the first event).

Where (tingly-box `b788a3b`):

- `internal/protocol/stream/anthropic_helper.go:75` (`SendStreamingError`)
  and `:88` (`SendForwardingError`): the body is
  `{"error":{"message":…,"type":"api_error"}}` whatever the status and the
  client: no Anthropic `{"type":"error",…}` wrapper, and `api_error` also
  for 429 (`rate_limit_error`), 400 (`invalid_request_error`), 401, 529.
- `internal/protocolserver/error_response.go:59` (`SendErrorResponse`): the
  body has `"type":"protocol_error"` (`:74`) and `"code"` set to tingly-box's
  own description of the call site (`:75`, e.g. `"Failed to forward
  request"`). The upstream's `type` and `code` survive only inside the
  message text.

Trigger: an upstream HTTP error, e.g. an OpenAI-style 429
`rate_limit_exceeded` to an Anthropic client, or a 400
`context_length_exceeded` to a Responses client.

Observed: Anthropic client:
`429 {"error":{"message":"Failed to forward request: Upstream error: 429 Too Many Requests (type=requests, code=rate_limit_exceeded): …","type":"api_error"}}`.
Responses client:
`400 {"error":{"message":"Upstream error: 400 Bad Request (type=invalid_request_error, code=context_length_exceeded): …: Failed to forward request","type":"protocol_error","code":"Failed to forward request"}}`.

Expected: the error in the client's protocol: for an Anthropic client
`{"type":"error","error":{"type":"rate_limit_error","message":…}}` (type
from the status); for an OpenAI client the upstream's `type` and `code`
(`invalid_request_error`, `context_length_exceeded`) when the upstream is
OpenAI-style, and a code mapped from the status otherwise.

Realism: Codex auto-compacts its history when a request fails with
`code: "context_length_exceeded"`; behind tingly-box it gets
`code: "Failed to forward request"` and the session stops instead.
Anthropic SDK clients get an error body of the wrong shape and type
(`api_error` for a rate limit). The status code itself is propagated
correctly.

Status: **new**. Related: #2052 (an error inside a 200 stream surfaces as
500), a different bug on the same error paths.

How it was found: by hand (code reading, round 2), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_GatewayErrorIsAnthropicShaped` and
`TestGo2LeanRepro_GatewayErrorCodeReachesResponsesClient`
(`repro/internal/protocolserver/zz_go2lean_repro_hunt2_test.go`, through the
gateway).

## General property

For every upstream error response, the error body the client receives has
the shape of the client's protocol (Anthropic:
`{"type":"error","error":{"type","message"}}` with the type that matches
the status; OpenAI Chat/Responses: `{"error":{"message","type","code"}}`),
and the upstream's error type and code are kept whenever the client's
protocol can express them.

Target functions: `protocolserver.SendErrorResponse` (and
`(*ProtocolHandler).failForward`), `stream.SendForwardingError`,
`stream.SendStreamingError`.
