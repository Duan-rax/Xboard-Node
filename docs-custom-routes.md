# Custom Routes

## Quick Example

```json
{
  "custom_route_rules": [
    {
      "name": "direct-example",
      "match": {"domain_suffixes": ["example.com"]},
      "action": {"type": "direct"}
    },
    {
      "name": "block-ads",
      "match": {"domains": ["ads.example.com"]},
      "action": {"type": "block"}
    },
    {
      "name": "route-warp",
      "match": {"ip_cidrs": ["1.1.1.0/24"], "ports": ["80", "443"]},
      "action": {"type": "route", "target": "warp-out"}
    }
  ]
}
```

## Match Conditions

| Condition | Description | Example |
|-----------|-------------|---------|
| `domains` | Exact domain match | `["api.example.com"]` |
| `domain_suffixes` | Suffix match | `["example.com"]` |
| `domain_keywords` | Domain substring match | `["tracker"]` |
| `domain_regexes` | RE2-compatible domain expression | `["(^|\\\\.)cdn[0-9]+\\\\.example$"]` |
| `ip_cidrs` | IP CIDR ranges | `["10.0.0.0/8"]` |
| `ports` | Port (single or range) | `["443", "8000-9000"]` |
| `networks` | Protocol | `["tcp"]` or `["udp"]` |
| `source_cidrs` | Source IP CIDR | `["192.168.1.0/24"]` |
| `source_ports` | Source port | `["1024-65535"]` |
| `protocols` | Sniffed application protocol | `["http", "tls", "quic", "bittorrent"]` |
| `geoip` | GeoIP collection | `["cn"]` |
| `geosite` | GeoSite collection | `["category-ads-all"]` |
| `rule_sets` | Native sing-box rule-set tag | `["my-local-set"]` |

## Action Types

| Action | Description |
|--------|-------------|
| `{"type": "direct"}` | Direct connection, bypass proxy |
| `{"type": "block"}` | Block connection |
| `{"type": "route", "target": "tag"}` | Route to specified outbound (by tag) |

## Application Order

1. Structured `custom_route_rules` (highest priority)
2. Raw `custom_routes`
3. Built-in blocklist rules
4. Panel routes, ordered by ascending `priority`

## Panel Route Schema 2

Xboard route groups can now be sent in the legacy `routes` field with a
structured `criteria` object:

```json
{
  "id": 9,
  "match": [],
  "criteria": {
    "domains": ["api.example.com"],
    "ports": ["443", "8000-8100"],
    "networks": ["tcp"]
  },
  "match_mode": "all",
  "priority": 20,
  "disabled": false,
  "action": "proxy",
  "action_value": "warp-out"
}
```

- `match_mode: all` produces one rule containing all supplied matcher fields.
- `match_mode: any` produces separate rules for each matcher category.
- `disabled` rules are ignored.
- Routes without `criteria` continue to use the legacy `match` array.

Within each category the supplied values are ORed. In `all` mode different
categories are ANDed. Domain and sniffed-protocol rules enable traffic sniffing
automatically.

## Kernel Compatibility

| Feature | Xray | Sing-box |
|---------|------|----------|
| All match conditions | ✅ | ✅ |
| direct / block / route | ✅ | ✅ |
| GeoIP / GeoSite | Native `geoip:` / `geosite:` | Remote SRS `rule_set` |
| Native `rule_sets` tags | ❌ | ✅ |

Sing-box no longer accepts the legacy `geoip` and `geosite` route fields.
Automatic Geo collections are compiled to remote `.srs` rule-sets and cached
through `experimental.cache_file`.

## Geo Data Sources

Geo databases are downloaded on each backend node, not on the panel host.
The default source is MetaCubeX.

```yaml
kernel:
  type: singbox
  geo_data_dir: /etc/xboard-node
  geo_rule_set_provider: metacubex # metacubex | sagernet | custom
  geo_data_variant: full           # full | lite

  # Xray .dat/.db provider. SagerNet falls back to Loyalsoldier for Xray.
  geo_data_provider: metacubex # metacubex | loyalsoldier | sagernet | custom
```

Custom rule-set sources use `{tag}` in the URL template:

```yaml
kernel:
  geo_rule_set_provider: custom
  geoip_rule_set_url: https://rules.example/geoip/{tag}.srs
  geosite_rule_set_url: https://rules.example/geosite/{tag}.srs
```

Downloaded SRS files are stored under
`geo_data_dir/rulesets/<provider>/<variant>/<geoip|geosite>/`. The node tries
jsDelivr first, falls back to GitHub, keeps a valid cached file if an update
fails, and refreshes rule-sets daily.

## Best Practices

- **Prefer** `custom_route_rules`: cross-kernel compatible, panel-managed
- **Use** `custom_routes` only: when native features are needed (e.g., load balancing)
