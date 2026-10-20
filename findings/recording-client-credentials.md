# Finding 12: Recordings keep the request URL's query string (new, not reported; minor; scope corrected)

Severity: **low** (see [the findings table](README.md)); recorded first as
medium, when the page said that every recording keeps the client's
`Authorization` / `X-Api-Key` headers. The repro test showed that the
headers do not reach the file on the gateway path; only the query string
does.

Path: recording (a rule or scenario with `recording` set to a mode that
includes `client_request`, e.g. `staged_request_response`), Anthropic,
Chat and Responses routes.

Where (tingly-box `b788a3b`, unchanged on main `bc460f6`):

- `internal/recording/recorder.go:64-69` (`NewProtocolRecorder`) stores the
  inbound request as `URL: c.Request.URL.String()` and
  `Headers: headerToMap(c.Request.Header)`, without redaction.
- `internal/protocolserver/protocol_transform.go:244-246` adds the
  pre-transform recorder (`StagePre`) to every chain that records the client
  request; `internal/protocolserver/recording_transform.go:76-99`
  (`toRecordRequest`) replaces the stored request with one whose headers
  are an empty map but whose URL is again `c.Request.URL.String()`, query
  string included. The upstream request (`StagePost`) gets the same URL.
- Nothing in `internal/obs` redacts the record before
  `GzipFileExporter` writes it (`internal/obs/gzip_exporter.go`).

So the client's headers reach a recording only when the chain fails before
the `StagePre` recorder runs (the record is then emitted by `RecordError`
with the headers as received); none of the transforms before it returns an
error today, so this is latent. The query string reaches every recording.

Trigger: a recorded request with a key in the query string
(`/tingly/anthropic/v1/messages?key=…`).

Observed: the recording's `original_request.url` (and
`transformed_request.url`) is `/tingly/anthropic/v1/messages?key=…`;
`original_request.headers` is empty.

Expected: credential-like query parameters (and, for the error path,
`Authorization`, `X-Api-Key`, `Cookie`) redacted, as `ai/quota/httplog.go`
already does for headers in its own logs.

Realism: low. The gateway's model routes do not read credentials from the
query string, so only a client configured with a Gemini-style `?key=` base
URL would put one there. Recordings are a debugging aid that users attach
to bug reports, so what they keep matters, but the header leak this page
first described does not happen on the gateway path.

Status: **new, minor** (not reported; tracker search 2026-10-20 over all
issues and PRs for `redact`, `recording` with `credential`, `api key`,
`authorization`: nothing). Reproduces on main `bc460f6` (query string
only).

How it was found: credentials direction, by reading; the repro test was
written afterwards (2026-10-20) and corrected the scope.

Evidence: `TestGo2LeanRepro_RecordingRedactsClientCredentials`
(`repro/internal/protocolserver/zz_go2lean_repro_rec_test.go`): runs the
gateway with a recording rule and the file-backed sink, sends a request
with the token in `x-api-key` and `Authorization`, a cookie and a
`?key=` query parameter, reads the `.jsonl.gz` recording back and fails on
`b788a3b` and `bc460f6` with "recording keeps the client's query-string key
in clear" (the header checks pass).
