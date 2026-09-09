# Xboard 流量审计：测试机从零部署

适用：已有可用的 Xboard 面板、准备一台**空白 Linux 测试 VPS**。本教程从下载本 fork 的 Xboard-Node 后端开始，安装节点服务、绑定面板、添加审计、最后验收。该方案仅供管理员在后台查看连接的域名/IP、流量、连接数与可归属的用户 ID；不修改用户客户端或订阅，也不记录网页内容或 URL 路径。

> 以下所有 `<>` 是占位符，必须替换；不要把真实 token、密码或 secret 发到聊天记录。

## 0. 范围、系统要求与面板准备

本教程安装的是 **Xboard-Node 后端**，不是重新安装 Xboard 面板。你的现有面板负责创建“机器/节点”、签发 token、下发用户与节点配置；测试 VPS 运行 Xboard-Node、sing-box/Xray 和采集器。

测试 VPS 需要：Ubuntu/Debian/CentOS 系统、root 或 sudo、systemd、可出站访问 GitHub 与面板。先执行：

```bash
sudo -i
apt-get update && apt-get install -y curl ca-certificates
uname -m
```

在 Xboard 面板先完成：

1. 新建一台测试“机器”（Machine）或“节点”（Node）；推荐 Machine，与现有多实例配置一致。
2. 记录机器 ID 与机器 Token；Token 只在测试 VPS 输入一次，不能提交至 Git 或发给任何人。
3. 为测试机创建一个 sing-box 节点并确认节点协议、端口与证书设置完整。

## 1. 准备参数

在开始前准备：

| 名称 | 示例 |
| --- | --- |
| 面板地址 | `https://panel.example.com` |
| 面板安全路径 | `admin-safe-path` |
| 节点标识 | `test-sg-singbox-01` |
| 节点架构 | `amd64`（`uname -m` 为 `x86_64`）或 `arm64`（`aarch64`） |
| 审计签名密钥 | 用 `openssl rand -hex 32` 生成 |
| 本机 Clash API 密钥 | 用 `openssl rand -hex 32` 生成 |

## 2. 下载并安装本 fork 的 Xboard-Node 后端

以下命令将下载 **Duan-rax/Xboard-Node 的 dev Release**，而不是上游 cedar2025 版本；它包含 Reality 本地 `dest` 覆盖、`xver` 与 Xray access log 支持。

```bash
sudo -i
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

install -d -m 0700 /root/xboard-node-install
cd /root/xboard-node-install
curl -fL -o install.sh https://raw.githubusercontent.com/Duan-rax/Xboard-Node/dev/install.sh
curl -fL -o xboard-node "https://github.com/Duan-rax/Xboard-Node/releases/download/dev/xboard-node-linux-${ARCH}"
curl -fL -o xbctl "https://github.com/Duan-rax/Xboard-Node/releases/download/dev/xbctl-linux-${ARCH}"
chmod 700 install.sh xboard-node xbctl
./xboard-node -v
./xbctl version
```

输入面板信息并安装 Machine 模式。`read -s` 防止 Token 写入终端历史：

```bash
read -r -p 'Panel URL: ' PANEL_URL
read -r -p 'Machine ID: ' MACHINE_ID
read -r -s -p 'Machine token: ' PANEL_TOKEN; echo

bash ./install.sh --mode machine \
  --panel "$PANEL_URL" \
  --machine-id "$MACHINE_ID" \
  --token "$PANEL_TOKEN" \
  --kernel singbox \
  --binary ./xboard-node \
  --xbctl-binary ./xbctl
unset PANEL_TOKEN
```

若你在面板创建的是传统 Node 而不是 Machine，使用以下命令替换上面的安装命令：

```bash
read -r -p 'Panel URL: ' PANEL_URL
read -r -p 'Node ID: ' NODE_ID
read -r -s -p 'Node token: ' PANEL_TOKEN; echo

bash ./install.sh --mode node \
  --panel "$PANEL_URL" \
  --node-id "$NODE_ID" \
  --token "$PANEL_TOKEN" \
  --kernel singbox \
  --binary ./xboard-node \
  --xbctl-binary ./xbctl
unset PANEL_TOKEN
```

安装器会创建：

```text
/usr/local/bin/xboard-node    节点后端
/usr/local/bin/xbctl          管理命令
/etc/xboard-node/config.yml   节点配置
/etc/xboard-node/credentials.env  私有凭据（0600）
/etc/systemd/system/xboard-node.service
```

立即验证后端与服务：

```bash
xbctl version
xbctl list
xbctl status
systemctl status xboard-node --no-pager
journalctl -u xboard-node -n 100 --no-pager
```

只有这一步显示 `active (running)` 且面板节点在线时，才继续添加审计。

## 3. 面板：安装 Risk Audit 插件

1. 在 Xboard 后台“插件”上传 [RiskAudit-4.2.0.zip](../RiskAudit-4.2.0.zip)。
2. 安装并启用 `Risk Audit` / `risk_audit`。
3. 打开插件“配置”，设置：
   - `webhook_secret`：填审计签名密钥；
   - `retention_days`：测试建议 `7`，生产按最短必要周期设置；
   - `max_clock_skew_seconds`：保留 `300`。
4. 用现有 Xboard 管理员邮箱和密码登录插件页：

   ```text
   https://<面板域名>/<安全路径>/risk-audit/login
   ```

   这是一个普通路径，**不能**写作 `#/risk-audit`。Xboard 原生管理端的 `#` 是前端 HashRouter，与插件无关。

初次进入时没有数据是正常的；先完成节点采集器部署。

## 4. 测试 sing-box 节点

### 4.1 配置 Xboard-Node

编辑 `/etc/xboard-node/config.yml` 中该测试实例。YAML 缩进必须保留：

```yaml
instances:
  - id: <面板显示的实例 ID>
    panel:
      url: https://panel.example.com
    kernel:
      type: singbox
      custom_config: "/etc/xboard-node/singbox-audit.json"
      log_level: info
    config_dir: /etc/xboard-node/instances/<实例 ID>
    machine:
      machine_id: <机器 ID>
      token_env: <原有环境变量名称>
```

`log_level: info` 是用户归属的必要条件。它使 sing-box 在本机 journal 中留下认证 UUID、来源端口与目标地址，采集器用这三项关联每一条连接。

创建 `/etc/xboard-node/singbox-audit.json`：

```json
{
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090",
      "secret": "<本机 Clash API 密钥>"
    }
  }
}
```

`external_controller` 必须是 `127.0.0.1`，不可使用 `0.0.0.0`。

重启并确认内核可用：

```bash
xbctl restart
systemctl status xboard-node --no-pager
curl -fsS -H 'Authorization: Bearer <本机 Clash API 密钥>' http://127.0.0.1:19090/connections
```

最后一条应返回 JSON；若返回 401，通常是 `secret` 与请求头不一致。

### 4.2 安装审计采集器

```bash
ARCH=amd64  # aarch64 主机写 arm64
curl -fL -o /tmp/traffic-audit-collector "https://github.com/Duan-rax/Xboard-Node/releases/download/dev/traffic-audit-collector-linux-${ARCH}"
install -m 0755 /tmp/traffic-audit-collector /usr/local/bin/traffic-audit-collector
install -d -m 0700 /etc/traffic-audit-collector
```

创建 `/etc/traffic-audit-collector/config.json`：

```json
{
  "panel_event_url": "https://panel.example.com/api/v1/risk-audit/events",
  "webhook_secret": "<插件 webhook_secret>",
  "interval_seconds": 2,
  "targets": [
    {
      "node_id": "test-sg-singbox-01",
      "kernel": "singbox",
      "clash_api": "http://127.0.0.1:19090",
      "clash_secret": "<本机 Clash API 密钥>",
      "singbox_journal_unit": "xboard-node"
    }
  ]
}
```

创建 `/etc/systemd/system/traffic-audit-collector.service`：

```ini
[Unit]
Description=Traffic Audit Collector for Xboard
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/traffic-audit-collector -config /etc/traffic-audit-collector/config.json
Restart=always
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict

[Install]
WantedBy=multi-user.target
```

```bash
chmod 600 /etc/traffic-audit-collector/config.json
systemctl daemon-reload
systemctl enable --now traffic-audit-collector
systemctl status traffic-audit-collector --no-pager
```

### 4.3 验收

1. 用一个已知 Xboard 用户连接测试节点并访问几个 HTTPS 域名。
2. 节点执行：

   ```bash
   journalctl -u xboard-node -n 100 -o cat --no-pager
   journalctl -u traffic-audit-collector -n 100 --no-pager
   ```

3. 打开插件“流量审计”。新记录应出现目标域名和 `#用户ID`。
4. 在“流量分析”上方输入该用户 ID，点击“应用”；卡片、分布和事件应同步筛选。

旧事件或采集器启动前已经存在的连接可能没有用户归属；重新建立一个连接即可测试新链路。

## 5. 可选：测试 Xray Reality + Nginx 回落

此部分只用于 Xray 节点；对 sing-box 无影响。

在节点实例 `kernel` 下加入：

```yaml
type: xray
xray_reality_dest_override: "127.0.0.1:8001"
xray_reality_xver: 1
xray_access_log: "/var/log/xboard-node/xray-access.log"
```

面板 Reality 的 `dest` 和 `server_name` **仍填公网值**，例如 `sg.example.com:443` 与 `sg.example.com`。本 fork 仅在节点运行时将目标替换为本地 Nginx，因此订阅 SNI 正确，无需篡改 `/etc/hosts`。

Nginx：

```nginx
server {
    listen 127.0.0.1:8001 ssl proxy_protocol default_server;
    server_name _;
    ssl_reject_handshake on;
}

server {
    listen 127.0.0.1:8001 ssl proxy_protocol;
    server_name sg.example.com;
    set_real_ip_from 127.0.0.1;
    real_ip_header proxy_protocol;
    ssl_certificate     /etc/letsencrypt/live/sg.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/sg.example.com/privkey.pem;
    # 正常站点配置
}
```

`xver: 1` 会向本地 Nginx 发送 PROXY protocol。若 `dest` 不是你控制的 PROXY-protocol 服务，删除 `xray_reality_xver`。

把以下对象追加至采集器 `targets` 数组，随后重启服务：

```json
{
  "node_id": "test-xray-01",
  "kernel": "xray",
  "xray_access_log": "/var/log/xboard-node/xray-access.log"
}
```

```bash
install -d -m 0750 /var/log/xboard-node
nginx -t && systemctl reload nginx
xbctl restart
systemctl restart traffic-audit-collector
tail -f /var/log/xboard-node/xray-access.log
```

Xray 会记录用户与目标地址，但不能提供可信的逐域名上下行字节数；该指标只在 sing-box 采集链路中可用。

## 6. 停止测试

若不再需要采集，执行 `systemctl disable --now traffic-audit-collector` 即可停止采集，不会影响节点代理。审计数据会按插件的 `retention_days` 自动清理；如需提前删除，应先完成数据库备份并走单独的变更流程，避免误删其他 Xboard 数据。
