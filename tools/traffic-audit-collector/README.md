# Traffic Audit Collector

This sidecar turns node-side connection information into signed events for the `RiskAudit` Xboard plugin. It never changes client subscriptions or client configurations.

## Sing-box: full domain/IP connection accounting

For sing-box, the collector polls the local Clash API `/connections`. It sends per-connection start events and traffic deltas, so the Xboard administrator dashboard can aggregate destination, upload, download, and distinct connection count.

Add this to the **local** sing-box `custom_config` for the target Xboard-Node instance. Do not expose the API publicly.

```json
{
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090",
      "secret": "REPLACE_WITH_A_LONG_LOCAL_SECRET"
    }
  }
}
```

Set the matching endpoint and secret in `config.json`, then run the collector on the same server. The Clash API must listen on `127.0.0.1`; never use `0.0.0.0` for this purpose.

For reliable Xboard user attribution, set the sing-box instance to `kernel.log_level: info` and configure the collector target with `singbox_journal_unit: "xboard-node"`. The collector joins the authenticated UUID from sing-box journal lines to the same Clash connection by source IP:port and destination. This does not expose passwords.

Some sing-box Clash API builds omit the authenticated user from `/connections`. For Xboard-Node, set the node `kernel.log_level` to `info` and configure `singbox_journal_unit: "xboard-node"` in the collector target. The collector correlates sing-box's connection ID log lines (source, destination, authenticated UUID) with the Clash connection metadata, then forwards the UUID to Xboard for user-ID mapping. If a connection has no matching user log, it remains visible only in the all-user aggregate.

## Xray / V2bX: website records and connection counts

The collector tails a standard Xray access log and sends destination, source, and `user@<id>` when present. Xray access logs do not provide a trustworthy per-destination upload/download counter, so its events contribute website records and connection counts, but not per-site bytes.

For V2bX, point `xray_access_log` at its Xray access-log file after enabling access logging in V2bX.

For current Xboard-Node Xray, the generated config disables Xray access logs; `kernel.custom_config` cannot replace its generated `log` section. This package includes `patches/xboard-node-enable-xray-access-log.patch`, which adds `kernel.access_log` to the node configuration and passes it into the generated Xray config. Build a pinned Xboard-Node release with the patch, then configure the Xray instance as follows:

```yaml
kernel:
  type: xray
  access_log: /var/log/xboard-node/xray-access.log
```

Create the parent directory with permissions that permit the Xboard-Node service user to write and the collector service to read. Xray events contribute domain/IP, source, user ID when present, and connection count; the Xray access log has no per-destination byte counters.

## Build and run

```bash
go build -trimpath -ldflags="-s -w" -o traffic-audit-collector .
sudo install -m 0755 traffic-audit-collector /usr/local/bin/traffic-audit-collector
sudo install -d -m 0700 /etc/traffic-audit-collector
sudo install -m 0600 config.example.json /etc/traffic-audit-collector/config.json
sudo install -m 0644 traffic-audit-collector.service /etc/systemd/system/traffic-audit-collector.service
sudo systemctl daemon-reload
sudo systemctl enable --now traffic-audit-collector
```

Set `panel_event_url` to the Xboard panel endpoint and set `webhook_secret` to the same value configured in the RiskAudit plugin. Keep `config.json` mode `0600`.

## Retention and limits

This stores destination metadata, not encrypted page contents. Use a short retention period and an event ingestion rate appropriate for your node capacity. Very short-lived connections between polls can be missed; set `interval_seconds` to 1 for greater fidelity at higher panel write volume.
