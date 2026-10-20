# Finding 12: Recordings keep the client's credentials (new, not reported; minor)

`internal/recording/recorder.go:68` stores the inbound request as
`Headers: headerToMap(c.Request.Header)` and `URL: c.Request.URL.String()`
without redaction, and nothing in `internal/obs` or `internal/recording`
redacts later. Every recording file holds the client's `Authorization` /
`X-Api-Key` (the gateway's model token or a sharing key), cookies, and any
key passed in the query string. Recordings are a debugging aid that users
attach to bug reports; `ai/quota/httplog.go` already has a header redaction
list for its own logs.

How it was found: credentials direction, by reading; the code path is a
one-liner, no repro test yet.
