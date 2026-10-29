# Changelog

All notable fork changes are recorded here. Release dates use the
`Asia/Singapore` timezone.

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
