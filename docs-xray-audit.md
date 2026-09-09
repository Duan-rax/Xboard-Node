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
can contain the Xboard user ID, destination domain/IP, source and route result.

Configure the existing TrafficAuditCollector target to tail the same file:

```json
{
  "node_id": "xray-reality-sg",
  "kernel": "xray",
  "xray_access_log": "/var/log/xboard-node/xray-access.log"
}
```

This provides per-user connection events. Xray access logs do not provide
per-destination upload/download byte counters, so exact website byte analysis
remains available only through sing-box connection tracking.
