# Finding 35: Anthropic Beta metadata transform panics on a user_id without account (fixed upstream on the gateway path; latent)

Severity: **high** (see [the findings table](README.md)).

Path: Anthropic client (Beta, `/v1/messages?beta=true`) -> Anthropic
provider at `api.anthropic.com` / `claude.ai` with an API key (the vendor
transform's Claude Code backend branch, legacy identity, which is the
default).

Where (tingly-box `b788a3b`):

- `internal/protocol/ops/request_anthropic_model.go:363-369`
  (`ApplyAnthropicBetaMetadataTransform`): when the inbound
  `metadata.user_id` parses, it calls `m.Fix(extra)` (`:366`) with no
  `recover`. Its V1 twin, `ApplyAnthropicV1MetadataTransform`, wraps the same
  call in a `recover` (`:310-320`, `m.Fix` at `:317`).
- `internal/protocol/metaid/user_id.go:79-107` (`(*MetadataUserID).Fix`)
  panics with "missing account uuid" (`:99-100`) when neither the inbound
  user_id nor `extra["user_id"]` has an account, and with "missing device
  id" (`:88-89`) when neither has a device.
- `extra` comes from `configExtraForMetadata`
  (`internal/protocol/transform/chain.go:199`), which has `user_id` only
  when the provider is OAuth (`WithProvider`, `chain.go:54-64`). The
  transform runs from `internal/protocol/transform/vendor.go:124` for every
  Claude Code backend (`isClaudeCodeBackend`, `vendor.go:77-82`: host
  `api.anthropic.com` or `claude.ai`, or a Claude Code OAuth provider).
- The panic is caught by `gin.Recovery`
  (`internal/server/server_routes.go:17`): the client gets a 500.

Trigger: a Beta request whose `metadata.user_id` parses (JSON
`{"device_id":…,"account_uuid":"","session_id":…}` or the legacy
`user_<hex>_account__session_<uuid>`) with an empty account, sent to an
API-key provider at `api.anthropic.com`.

Observed: `ApplyAnthropicBetaMetadataTransform` panics with "missing account
uuid"; through the gateway, the request fails with 500 and never reaches
Anthropic.

Expected: the transform returns a request, as the V1 transform does (with
the user_id kept as sent, or rebuilt without an account).

Realism: high. Claude Code signed in with an API key (no claude.ai login)
sends exactly this user_id, with an empty `account_uuid`, on every request,
and it uses the Beta endpoint. Routed to an `api.anthropic.com` provider with
an API key, every request fails: the gateway is unusable for that pairing.
OAuth (Claude Code subscription) providers are not affected, since their
`Config.UserID` fills the account.

Status: **fixed upstream on the gateway path** in `9b15a58` (#2134, merged
2026-10-20, after `b788a3b`): the vendor transform now runs the metadata
transform only for Claude Code OAuth providers (`isClaudeCodeOAuth`,
`internal/protocol/transform/vendor.go` on main `bc460f6`), whose `extra`
always carries a `user_id`, so the API-key trigger above no longer reaches
it. `ApplyAnthropicBetaMetadataTransform` itself is unchanged and still
panics when called without an account (the repro test calls it directly and
still fails on `bc460f6`): latent. No issue about the panic (tracker search
2026-10-20 over the full text of all issues and PRs for `user_id`,
`account_uuid`, `panic`, `metadata`).

How it was found: by hand (code reading, round 4), not by a go2lean search,
like findings 6-11.

Evidence: `TestGo2LeanRepro_BetaMetadataEmptyAccountNoPanic`
(`repro/internal/protocol/ops/zz_go2lean_repro_hunt4_test.go`).

Fix direction: recover as the V1 transform does, or better, make `Fix`
return an error instead of panicking and keep the inbound user_id when it
fails.

## Further triggers

Found while triaging the M2 agent's run on `ops`. `ParseMetadataUserID`
accepts any JSON object, and `null`, so a `user_id` of `{}` (not only
Claude Code's `"account_uuid":""`) reaches the unguarded `m.Fix(extra)`
(`ops/request_anthropic_model.go:366`) and panics with "missing account
uuid". The other panic in `metaid.(*MetadataUserID).Fix`, "missing device
id" (`internal/protocol/metaid/user_id.go:79-107`), is not reachable from a
client: the gateway always fills `extra["device"]` (`config.go:335-341`
fills `ClaudeCodeDeviceID` when empty, `protocol_transform.go:129` and
`transform/chain.go:204-206` pass it on); only an operator blanking it in
a hot-reloaded config gets there.

## General property

For every Anthropic Beta request and every provider config (`extra` map),
`ApplyAnthropicBetaMetadataTransform` returns normally (no panic) with a
request; for the same inputs it behaves as `ApplyAnthropicV1MetadataTransform`
does on the V1 request.

Target function: `ops.ApplyAnthropicBetaMetadataTransform` (via the vendor
transform, `transform.(*VendorTransform).Apply` for an Anthropic Beta
request).
