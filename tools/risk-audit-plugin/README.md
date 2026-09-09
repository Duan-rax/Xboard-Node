# Risk Audit 4.2 — Xboard 管理员流量审计

这是仅面向 Xboard 管理员的流量分析与风险审计插件。页面参考“流量分析 / 流量审计记录”布局，提供时间范围、上传/下载/总流量/连接数卡片、服务或目标地址分布，以及按用户 ID、目标域名、规则标签和动作筛选的审计事件表。

它不会出现在用户前台，也不会记录网页内容；保存的是节点上报的风险事件、目标地址、规则标签、协议、用户 ID 与可选流量计数。

要获得“全量网站 + 上传/下载/连接数”数据，请同时部署本套件中的 `TrafficAuditCollector`。插件本身负责管理后台、数据库与聚合，不会自行读取节点流量。

流量分析页支持按用户 ID 查看，并会同时筛选汇总卡片、服务/目标地址分布与审计事件。该功能依赖采集器识别到用户：Xray 从 `user@<Xboard ID>` 访问日志取得归属；sing-box 从 `xboard-node` 的同一连接入站日志关联 UUID，再由插件映射为 Xboard 用户。未升级采集器或未开启 sing-box `log_level: info` 时，历史事件中的用户列会显示 `—`，这是数据尚未携带归属，不是筛选功能失效。

插件侧栏只包含实际可用的“流量分析”和“流量审计”两个页面，不包含 Xboard 主面板的仪表盘、系统、节点或路由占位入口。

## 安装

1. 将 `RiskAudit` 目录压缩为 zip，在 Xboard 管理后台的“插件”上传。
2. 安装并启用 `risk_audit`。
3. 在插件设置填入高随机度的 `webhook_secret`。
4. 在管理后台安全路径打开：

   ```text
   https://你的面板域名/管理后台安全路径/risk-audit
   ```

   Xboard 原生管理端使用 `#/config/plugin` 这类 HashRouter 地址；插件页必须使用上面的斜杠路径，**不要**写成 `#/risk-audit`。

首次打开会显示独立的管理员验证页。输入现有 Xboard 管理员邮箱与密码后，插件会校验该账号的 `is_admin` 状态，并建立自己的 HttpOnly 会话 Cookie。不会读取、要求或存储 Xboard API Token；普通用户无法进入页面或读取审计 API。

## 节点事件上报

### 通用 / sing-box 日志转发器

`POST /api/v1/risk-audit/events`

请求可使用 HMAC 签名：

```text
Content-Type: application/json
X-Audit-Timestamp: Unix epoch seconds
X-Audit-Signature: hex(HMAC-SHA256(timestamp + "." + raw request body, webhook_secret))
```

```json
{
  "event_id": "singbox-1720000000-000001",
  "user_id": 123,
  "node_id": "machine-10",
  "client_source": "tcp:203.0.113.10:49152",
  "occurred_at": "2026-09-08T12:00:00Z",
  "action": "block",
  "rule_tag": "block-bittorrent",
  "destination": "tracker.example.net:443",
  "destination_ip": "203.0.113.5",
  "network": "tcp",
  "protocol": "bittorrent",
  "upload_bytes": 0,
  "download_bytes": 0,
  "metadata": { "kernel": "sing-box" }
}
```

### Xray 路由 Webhook

Xray 支持为每条路由设置固定 HTTP 请求头。将下面的 `webhook` 填入 `xray-audit.json` 中需要审计的规则；`X-Audit-Rule` 为本规则的显示名称，`X-Audit-Node` 为节点标识。

```json
{
  "type": "field",
  "ruleTag": "block-streaming",
  "domain": ["domain:iqiyi.com"],
  "outboundTag": "block",
  "webhook": {
    "url": "https://PANEL_DOMAIN/api/v1/risk-audit/events",
    "deduplication": 60,
    "headers": {
      "X-Audit-Token": "WEBHOOK_SECRET",
      "X-Audit-Node": "machine-10",
      "X-Audit-Rule": "block-streaming"
    }
  }
}
```

Xray 原生 Webhook 不包含事件 ID；插件会使用 Webhook 请求体做去重，并自动把 Xboard-Node 的 `user@<id>` 映射为用户 ID。

## 数据保留

默认保留 30 天。插件每天 03:17 清理超过 `retention_days` 的事件。请设置为满足风险处置所需的最短周期，并向受影响用户履行适用的告知义务。

## 当前边界

- 面板插件无法自行读取节点流量，节点必须上报事件。
- Xray 规则 Webhook 适合记录特定风险规则命中；它不是完整浏览历史。
- HTTPS、ECH、直连 IP 与加密/混淆 P2P 可能无法提供目标域名。
- 当前 Xboard 管理后台前端为预编译 React 包，官方插件文档未提供稳定的侧栏 React 页面注入接口。因此插件使用后台安全路径下的独立管理页，并通过现有 Xboard 管理员账号建立插件会话。
