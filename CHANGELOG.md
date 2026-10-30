# Changelog

All notable fork changes are recorded here. Release dates use the
`Asia/Singapore` timezone.

## Unreleased

- Decode structured panel routes with `criteria`, `match_mode`, priority and
  disabled state.
- Compile `all` panel rules as one multi-condition rule and `any` rules as
  separate matchers for both Sing-box and Xray.
- Support domain keywords, RE2 domain expressions and sniffed application
  protocols in structured panel and custom routes.
- Generate current Sing-box routing actions and native `rule_set` definitions
  for GeoIP/GeoSite collections while keeping Xray on its native geo data.
- Preserve legacy `match` rules and apply panel routes in ascending priority
  order.
- Preserve panel `config_revision`/`config_hash` and return both in node status
  reports for reliable applied-config confirmation.
- Restore the last-known-good config/users when a reload and replacement kernel
  start both fail instead of leaving partially applied routing state.
- Limit automatic Sing-box sniffing to TCP to avoid an embedded AnyTLS/UDP
  packet nil-buffer crash while preserving UDP network/port/CIDR routing.
- Report the kernel name, effective embedded module version and structured
  route feature manifest in every node status update for panel capability
  negotiation.
- Emit a connection-level audit line containing only the numeric panel user ID,
  source, destination and sniffed domain so the native traffic-audit collector
  can attribute sing-box connections when the Clash API omits the user.
- Let the traffic-audit collector correlate AnyTLS user IDs by destination
  within a short TTL and associate Xray access-log records with panel users.
- Add machine-token audit bootstrap, automatic collector installation from
  GitHub releases, and one-command provisioning for new machines.
- Publish per-node Xray access-log targets through the machine audit manifest
  so newly added Xray nodes do not require collector reconfiguration.
- Split managed kernel generation into stable compiler boundaries:
  `route_compiler.go`, `outbound_compiler.go`, `reality_override.go` for Xray,
  and `route_compiler.go`, `outbound_compiler.go`, `rule_set_compiler.go` for
  Sing-box. Kernel `config.go` files now retain inbound/config assembly, custom
  config merging and stable call sites instead of all managed-rule logic.

### Fixed

- Keep Sing-box `all` matcher groups logically ANDed even when multiple domain
  or rule-set collections are present.
- Let custom Sing-box rule-set and experimental settings override generated
  values without dropping the cache needed by remote rule-sets.

### Geo data

- Default both kernels to MetaCubeX, with optional SagerNet and custom URL
  sources.
- Download sing-box SRS files on each backend node and cache them locally with
  CDN-to-GitHub fallback and daily refresh.
- Download Xray GeoIP/GeoSite databases on the backend node and point
  `XRAY_LOCATION_ASSET` at the node-owned directory.

## 2026.10.30.2

Release: `v2026.10.30.2`

- Switched official binary and container releases to tag-triggered GitHub
  Actions.
- Release tags now build amd64/arm64 binaries, upload checksums, publish a
  multi-architecture GHCR image and create the GitHub Release automatically.
- Go builders now run on the native `BUILDPLATFORM` and cross-compile for
  `TARGETARCH`, avoiding stalled arm64 builds under QEMU.
- Branch pushes remain verification-only and do not replace `latest`.

## 2026.10.30

Release: `v2026.10.30`

### Added

- Administrator-managed egress profiles for SOCKS5, HTTP/HTTPS,
  Shadowsocks, VLESS and Trojan.
- Batch node binding, rebinding, selected unbind and unbind-all workflows.
- Applied-config acknowledgement using `egress_proxy_id` and
  `config_applied_at`.
- Low-frequency REST reconciliation while retaining WebSocket configuration
  pushes.
- Dynamic Sing-box egress outbound replacement and catch-all route hot reload.
- Native amd64 and arm64 release binaries, `xbctl`, and traffic-audit
  collector artifacts.

### Fixed

- A missed WebSocket configuration event no longer leaves a node indefinitely
  on stale egress configuration.
- Egress changes no longer restart the kernel or disconnect all clients.
- Configuration metrics no longer race with status reporting.
- Stale egress outbounds are removed after the routing rules switch.

### Compatibility

- Requires the coordinated Xboard panel release `v2026.10.30` for the managed
  egress page and acknowledgement APIs.
- `custom_config` remains an advanced escape hatch. Structured panel routes
  take priority through their existing rule order; the managed egress is a
  final catch-all only after the configured custom and panel rules.

### Verification

- `go test ./internal/...` passed.
- Panel and node were deployed to production.
- All 13 production nodes report `d0c9dd0` or the release rebuild as active.
- Japan 01 gray test switched `direct -> Legend SG -> direct` without changing
  the node PID or dropping the control-plane WebSocket.
