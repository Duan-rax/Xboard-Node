# Xray audit access log

Set a local access-log path on an Xray instance to make its connection records
available to an audit collector:

```yaml
kernel:
  type: xray
  xray_access_log: "/var/log/xboard-node/xray-access.log"
```

The generated Xray configuration writes its `log.access` field to that path.
For VLESS users, Xboard-Node assigns `email: user@<ID>`, so Xray access records
contain the Xboard user ID, destination domain/IP, source and route result.

## Connection byte counters

Upstream Xray currently exposes cumulative per-user traffic through the
`user>>>email>>>traffic>>>uplink/downlink` stats, not bytes for an individual
connection. Xboard-Node therefore builds against a pinned Cedar commit with the
unmerged Xray PR
[#6843](https://github.com/XTLS/Xray-core/pull/6843) applied locally.

The resulting access line is emitted when the connection ends and includes:

```text
from 203.0.113.8:52144 accepted tcp:example.com:443 [inbound >> direct] email: user@7 up=1024 down=8192
```

`tools/prepare-xray-core.sh` downloads the exact Cedar source commit, applies
`tools/patches/xray-access-traffic.patch`, and creates an alternate Go modfile
that points to the patched source. The script also refreshes the alternate
module checksums from the local Xray dependency graph. Build commands use:

```bash
./tools/prepare-xray-core.sh
go test -modfile=.xray-audit/go.mod ./internal/...
go build -modfile=.xray-audit/go.mod -o xboard-node ./cmd/xboard-node
```

The patch has two operational consequences:

- Bytes are available only after a connection closes; long-lived connections
  produce their access record at teardown.
- Interrupted transfers report the bytes observed before teardown and may be
  incomplete.

Configure the TrafficAuditCollector target to tail the same file:

```json
{
  "node_id": "xray-reality-sg",
  "kernel": "xray",
  "xray_access_log": "/var/log/xboard-node/xray-access.log"
}
```

The collector recognizes both patched lines with `up=<bytes> down=<bytes>` and
legacy lines without byte counters. Legacy lines still provide connection and
user audit metadata, but their connection byte fields remain zero.
