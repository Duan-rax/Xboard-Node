# Xboard 流量审计：现有节点迁移指南

适用：已有 Xboard-Node / sing-box / Xray 节点在运行，希望不中断订阅、节点账号或现有自定义出站的前提下加入管理员审计。

策略：先备份；每次仅改一项；先验证 sing-box，再迁移 Xray；任一异常立即恢复备份并重启。

## 1. 迁移前检查和备份

在每台节点执行：

```bash
set -eu
stamp=$(date +%Y%m%d-%H%M%S)
install -d -m 0700 "/root/xboard-audit-backup-${stamp}"
cp -a /etc/xboard-node/config.yml "/root/xboard-audit-backup-${stamp}/config.yml"
[ -f /etc/xboard-node/singbox-audit.json ] && cp -a /etc/xboard-node/singbox-audit.json "/root/xboard-audit-backup-${stamp}/singbox-audit.json" || true
systemctl cat xboard-node > "/root/xboard-audit-backup-${stamp}/xboard-node.service.txt"
xbctl version || true
uname -m
```

不要覆盖原有 `custom_config`。若已有 `singbox-audit.json`，只在它的根对象中合并 `experimental.clash_api`；原有的 `outbounds`、`route`、`dns` 和审计规则必须保留。

## 2. 先升级现有 Xboard-Node 后端

此步适用于**所有**既有节点（sing-box 与 Xray）。升级会替换 `/usr/local/bin/xboard-node` 和 `xbctl`，但不会重建或覆盖 `/etc/xboard-node/config.yml`、实例目录或凭据。安装器会自动在 `/etc/xboard-node/backups/` 创建可回滚备份。

```bash
sudo -i
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

install -d -m 0700 /root/xboard-node-upgrade
cd /root/xboard-node-upgrade
curl -fL -o install.sh https://raw.githubusercontent.com/Duan-rax/Xboard-Node/dev/install.sh
curl -fL -o xboard-node "https://github.com/Duan-rax/Xboard-Node/releases/download/dev/xboard-node-linux-${ARCH}"
curl -fL -o xbctl "https://github.com/Duan-rax/Xboard-Node/releases/download/dev/xbctl-linux-${ARCH}"
chmod 700 install.sh xboard-node xbctl
./xboard-node -v
./xbctl version
bash ./install.sh upgrade --binary ./xboard-node --xbctl-binary ./xbctl
xbctl status
systemctl status xboard-node --no-pager
```

升级后确认现有用户仍可连通，再执行审计配置。若后端无法启动，安装器通常会自动恢复；仍异常时可用 `/etc/xboard-node/backups/` 中最新备份恢复，再 `systemctl restart xboard-node`。

## 3. 迁移现有 sing-box 节点

### 3.1 先确认节点当前健康

```bash
systemctl is-active xboard-node
xbctl restart
journalctl -u xboard-node -n 50 --no-pager
```

确认用户仍可连接后，再继续。

### 3.2 最小改动 config.yml

以现有实例为基础，只补充或修改这两项：

```yaml
kernel:
  type: singbox
  custom_config: "/etc/xboard-node/singbox-audit.json"
  log_level: info
```

不要更改 `id`、`machine_id`、`token_env`、`panel.url`、`config_dir`。`log_level: info` 会增加日志量，但只用于在本机将 UUID 对应到连接；建议同时配置 journald 日志轮转。

### 3.3 合并本机 Clash API

如果文件不存在，创建 `/etc/xboard-node/singbox-audit.json`：

```json
{
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090",
      "secret": "<新生成的本机密钥>"
    }
  }
}
```

如果已有文件，不要直接覆盖。用编辑器在其顶级 JSON 对象加入同样的 `experimental.clash_api`；若已有 `experimental`，只加入其 `clash_api` 子项。

重启并测试本地 API：

```bash
xbctl restart
curl -fsS -H 'Authorization: Bearer <本机密钥>' http://127.0.0.1:19090/connections
```

失败时先恢复备份的 `config.yml` / `singbox-audit.json`，再 `xbctl restart`。

### 3.4 无 Go 安装采集器

```bash
ARCH=amd64  # aarch64 使用 arm64
curl -fL -o /tmp/traffic-audit-collector "https://github.com/Duan-rax/Xboard-Node/releases/download/dev/traffic-audit-collector-linux-${ARCH}"
install -m 0755 /tmp/traffic-audit-collector /usr/local/bin/traffic-audit-collector
install -d -m 0700 /etc/traffic-audit-collector
```

若旧版采集器服务此前失败（例如曾显示 `go: command not found`），先停用旧服务：

```bash
systemctl disable --now traffic-audit-collector 2>/dev/null || true
```

写入配置：

```json
{
  "panel_event_url": "https://<面板域名>/api/v1/risk-audit/events",
  "webhook_secret": "<与面板插件完全一致的密钥>",
  "interval_seconds": 2,
  "targets": [
    {
      "node_id": "<此节点唯一标识>",
      "kernel": "singbox",
      "clash_api": "http://127.0.0.1:19090",
      "clash_secret": "<本机 Clash API 密钥>",
      "singbox_journal_unit": "xboard-node"
    }
  ]
}
```

使用从零教程中的相同 systemd unit，执行：

```bash
chmod 600 /etc/traffic-audit-collector/config.json
systemctl daemon-reload
systemctl enable --now traffic-audit-collector
journalctl -u traffic-audit-collector -n 100 --no-pager
```

## 4. 迁移现有 Xray Reality 节点

只有需要 Reality 本地 Nginx 回落、`xver` 或 Xray 用户审计时才更新 Xboard-Node 二进制。

### 4.1 仅新增本地运行时字段

面板里的 Reality `dest` / `server_name` 保持公网 SNI，例如 `sg.example.com:443` / `sg.example.com`。在节点 `/etc/xboard-node/config.yml` 的 Xray 实例加入：

```yaml
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8001"
  xray_reality_xver: 1
  xray_access_log: "/var/log/xboard-node/xray-access.log"
```

端口 `8001` 只是例子，可以替换为任意本机 Nginx/Caddy PROXY-protocol 监听端口。不要把 `127.0.0.1:端口` 填回面板 Reality `dest`；这是运行时覆盖字段的用途。

Nginx 至少要有一个拒绝未知 SNI 的默认 server，并为真实域名启用 `proxy_protocol`。验证后再重启：

```bash
install -d -m 0750 /var/log/xboard-node
nginx -t
systemctl reload nginx
xbctl restart
journalctl -u xboard-node -n 100 --no-pager
```

### 4.2 为同一个采集器增加 Xray target

在既有 `/etc/traffic-audit-collector/config.json` 的 `targets` 数组中追加：

```json
{
  "node_id": "<此 Xray 节点唯一标识>",
  "kernel": "xray",
  "xray_access_log": "/var/log/xboard-node/xray-access.log"
}
```

重启并检查：

```bash
systemctl restart traffic-audit-collector
tail -f /var/log/xboard-node/xray-access.log
```

有效 VLESS 连接的 access log 应出现 `email: user@<数字>`。Nginx 收到的 Reality 回落/伪装站流量不属于已认证的 Xboard 用户，不应期待它有用户 ID。

## 5. 迁移验收与回滚

| 检查 | 预期结果 |
| --- | --- |
| `systemctl is-active xboard-node` | `active` |
| Clash API | 仅 `127.0.0.1:19090` 可访问，使用 Bearer secret 返回 JSON |
| `traffic-audit-collector` | `active (running)`，日志没有 401、DNS 或连接错误 |
| 插件审计表 | 新 sing-box 记录有域名、流量和用户 ID |
| 用户筛选 | 输入用户 ID 后汇总、分布、审计记录同时变化 |
| Xray access log | 新连接出现目标与 `user@ID` |

任一节点异常时，先停采集器（不会影响代理），再恢复该节点的备份配置：

```bash
systemctl stop traffic-audit-collector
cp -af /root/xboard-audit-backup-<时间戳>/config.yml /etc/xboard-node/config.yml
[ -f /root/xboard-audit-backup-<时间戳>/singbox-audit.json ] && cp -af /root/xboard-audit-backup-<时间戳>/singbox-audit.json /etc/xboard-node/singbox-audit.json || true
xbctl restart
```

代理恢复后，再单独分析采集器日志；采集器故障不应阻断节点转发。
