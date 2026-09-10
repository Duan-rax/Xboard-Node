# Machine 模式：面板节点自动审计

此方案适用于一台 Xboard Machine 在面板中挂多个节点。Xboard-Node 会动态发现这些节点；管理员**不需要**在采集器里逐个填写 Node ID、Clash 端口或 secret。

## 工作方式

1. 面板把 Node 绑定到同一台 Machine。
2. Xboard-Node 为每个运行中的 sing-box Node 建立仅本机可访问的 Clash API。
3. 端口为 `singbox_audit_api_base_port + 面板 Node ID`。
4. Xboard-Node 以 `0600` 原子写入 manifest；采集器每五秒读取它。
5. 面板增加/移除 Node 后，Machine 自动发现并更新 manifest，采集器自动开始/停止该 Node 的轮询。

`node_id` 仍会出现在面板审计记录中，但由 Xboard-Node 自动从面板 Node ID 生成；不由管理员在 collector 配置中填写。

## 1. 配置 Machine 后端

在对应 Machine 实例的 `/etc/xboard-node/config.yml` 中设置一次：

```yaml
instances:
  - id: panel-example-machine-4
    panel:
      url: https://panel.example.com
    kernel:
      type: singbox
      log_level: info
      singbox_audit_api_enabled: true
      singbox_audit_api_base_port: 19090
      singbox_audit_api_secret: "替换为至少 32 字节的随机本机密钥"
      audit_manifest_path: "/etc/xboard-node/traffic-audit-targets.json"
    machine:
      machine_id: 4
      token_env: INSTANCE_PANEL_EXAMPLE_MACHINE_4_MACHINE_TOKEN
```

生成密钥：

```bash
openssl rand -hex 32
```

`log_level: info` 是将 sing-box 的认证 UUID 与连接关联到 Xboard 用户的必要条件。

选择基准端口时必须满足：

```text
基准端口 + 当前及未来最大的面板 Node ID <= 65535
```

例如基准端口 `19090`、面板 Node ID `33` 时，端口为 `19123`。这些 API 强制监听 `127.0.0.1`，不能对公网开放。

重启后端，待面板节点被发现：

```bash
xbctl service restart
journalctl -u xboard-node -n 100 --no-pager
cat /etc/xboard-node/traffic-audit-targets.json
```

该 manifest 含有本机 API secret，只应由 root 读取；不要粘贴或上传其内容。

## 2. 配置一个采集器

采集器配置不填 sing-box `targets`，只填写 manifest：

```json
{
  "panel_event_url": "https://panel.example.com/api/v1/risk-audit/events",
  "webhook_secret": "RiskAudit 插件的 webhook_secret",
  "interval_seconds": 2,
  "manifest_paths": [
    "/etc/xboard-node/traffic-audit-targets.json"
  ]
}
```

同一台服务器若运行多个独立 Machine 实例，为每个实例设置不同的 `audit_manifest_path`，并将这些路径全部列入 `manifest_paths`。

安装/更新 collector 后：

```bash
systemctl restart traffic-audit-collector
systemctl status traffic-audit-collector --no-pager
journalctl -u traffic-audit-collector -n 100 --no-pager
```

## 3. 在面板新增节点后的验收

1. 在 Xboard 面板把新 Node 绑定到该 Machine。
2. 等待 Machine 的同步周期或触发面板同步。
3. 检查 manifest 出现新 `panel_node_id`。
4. 用用户连接该 Node 并访问网站。
5. 在 RiskAudit 的流量审计表确认新事件有正确的节点、目标地址和用户 ID。

无需改 collector 配置、无需重启 collector，也无需为面板 Node 单独创建 JSON 文件。

## Xray

Xray 不使用 Clash API。Machine 自动清单只包含 sing-box Node；Xray 网站连接审计仍通过 `xray_access_log` 的静态 collector target 完成。
