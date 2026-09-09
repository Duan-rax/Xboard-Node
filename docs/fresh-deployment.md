# Xboard-Node 全新部署

本文适用于一台尚未安装 Xboard-Node 的 Linux VPS，目标是从本仓库的 `dev` Release 安装节点后端并连接现有 Xboard 面板。

本文不包含流量采集、访问审计、RiskAudit 插件或 Clash API。

> 文中的 `<...>` 是占位符。不要把面板 Token 提交到 Git 或发送到公开聊天。

## 1. 环境要求

- Debian、Ubuntu 或其他使用 systemd 的 Linux
- root 权限
- 可访问 GitHub、Xboard 面板和节点所需端口
- 支持 `amd64` 或 `arm64`

安装基础工具：

```bash
sudo -i
apt-get update
apt-get install -y curl ca-certificates
```

## 2. 下载本仓库的 Release

下面只把候选文件下载到临时安装目录，尚未修改系统中的服务。

```bash
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

INSTALL_DIR=/root/xboard-node-install
install -d -m 0700 "$INSTALL_DIR"
cd "$INSTALL_DIR"

RELEASE_BASE=https://github.com/Duan-rax/Xboard-Node/releases/download/dev

curl -fL --retry 3 -o install.sh https://raw.githubusercontent.com/Duan-rax/Xboard-Node/dev/install.sh
curl -fL --retry 3 -o xboard-node "$RELEASE_BASE/xboard-node-linux-${ARCH}"
curl -fL --retry 3 -o xbctl "$RELEASE_BASE/xbctl-linux-${ARCH}"

chmod 700 install.sh xboard-node xbctl
./xboard-node -v
./xbctl version
```

若要使用 Reality 本地回落功能，在安装前确认候选二进制包含相关配置字段：

```bash
for key in xray_reality_dest_override xray_reality_xver; do
  if grep -aq "$key" ./xboard-node; then
    echo "$key: present"
  else
    echo "$key: MISSING"
    exit 1
  fi
done

sha256sum ./xboard-node ./xbctl
```

`./xboard-node -v` 只验证当前目录中的候选文件，并不代表该文件已经安装。

## 3. 安装

### Machine 模式

先在面板创建 Machine，并记录 Machine ID 和 Token：

```bash
cd /root/xboard-node-install

read -r -p 'Panel URL: ' PANEL_URL
read -r -p 'Machine ID: ' MACHINE_ID
read -r -s -p 'Machine token: ' PANEL_TOKEN
echo

INSTALL_ARGS=(
  --mode machine
  --panel "$PANEL_URL"
  --machine-id "$MACHINE_ID"
  --token "$PANEL_TOKEN"
  --kernel xray
  --binary "$PWD/xboard-node"
  --xbctl-binary "$PWD/xbctl"
)
bash ./install.sh install "${INSTALL_ARGS[@]}"

unset PANEL_TOKEN
```

如需 sing-box，把 `--kernel xray` 改为 `--kernel singbox`。

### Node 模式

传统单节点使用：

```bash
cd /root/xboard-node-install

read -r -p 'Panel URL: ' PANEL_URL
read -r -p 'Node ID: ' NODE_ID
read -r -s -p 'Node token: ' PANEL_TOKEN
echo

INSTALL_ARGS=(
  --mode node
  --panel "$PANEL_URL"
  --node-id "$NODE_ID"
  --token "$PANEL_TOKEN"
  --kernel xray
  --binary "$PWD/xboard-node"
  --xbctl-binary "$PWD/xbctl"
)
bash ./install.sh install "${INSTALL_ARGS[@]}"

unset PANEL_TOKEN
```

必须保留 `--binary` 和 `--xbctl-binary`。它们明确要求安装器使用刚下载并验证过的本仓库文件。

## 4. 安装验收

服务真正运行的文件是 `/usr/local/bin/xboard-node`。候选文件、已安装文件和运行中进程必须一致：

```bash
PID=$(pidof xboard-node)

systemctl is-active xboard-node
systemctl show xboard-node -p ExecStart --no-pager
readlink -f "/proc/$PID/exe"

sha256sum /root/xboard-node-install/xboard-node /usr/local/bin/xboard-node "/proc/$PID/exe"

cmp -s /root/xboard-node-install/xboard-node /usr/local/bin/xboard-node
cmp -s /usr/local/bin/xboard-node "/proc/$PID/exe"
echo "candidate, installed binary and running process are identical"

xbctl status
journalctl -u xboard-node -n 100 --no-pager
```

如果任一 `cmp` 没有执行到最后的成功提示，安装或重启没有完成。不要继续改配置，先参照升级文档重新替换二进制。

## 5. 可选：Xray Reality 本地 Nginx 回落

面板中的 Reality `dest` 和 `server_name` 仍填写公网域名，例如：

```text
dest: test.example.com:443
server_name: test.example.com
```

只在节点的 `/etc/xboard-node/config.yml` 对应实例中加入本地运行时覆盖：

```yaml
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8001"
  xray_reality_xver: 1
```

`xray_reality_xver: 1` 表示 Xray 向本地回落服务发送 PROXY protocol v1。因此 Nginx 的监听项必须包含 `proxy_protocol`：

```nginx
server {
    listen 127.0.0.1:8001 ssl proxy_protocol default_server;
    server_name _;
    ssl_reject_handshake on;
}

server {
    listen 127.0.0.1:8001 ssl proxy_protocol;
    server_name test.example.com;

    set_real_ip_from 127.0.0.1;
    real_ip_header proxy_protocol;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    root /var/www/test.example.com;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}
```

应用配置：

```bash
nginx -t
systemctl reload nginx
xbctl restart
```

先直接验证 Nginx：

```bash
curl --haproxy-protocol --resolve test.example.com:8001:127.0.0.1 -kiv https://test.example.com:8001/
```

再验证完整的 `443 → Xray Reality → Nginx` 链路：

```bash
curl --resolve test.example.com:443:127.0.0.1 -kiv https://test.example.com/
```

返回站点内容或预期的 HTTP 状态码即表示回落链路可用。

## 6. 常见问题

### 下载成功，但运行中仍是旧版本

下载命令只写入 `/root/xboard-node-install`。必须继续执行安装命令，并在安装后比较三个文件的 SHA-256。

### Reality 配置字段没有生效

检查已安装文件和运行中进程，而不是只检查下载目录：

```bash
PID=$(pidof xboard-node)

for file in /usr/local/bin/xboard-node "/proc/$PID/exe"; do
  echo "=== $file ==="
  for key in xray_reality_dest_override xray_reality_xver; do
    grep -aq "$key" "$file" && echo "$key: present" || echo "$key: MISSING"
  done
done
```

### 浏览器显示 `ERR_INVALID_RESPONSE`

依次检查：

1. Nginx 是否监听 `127.0.0.1:8001`。
2. Nginx 的监听项是否包含 `ssl proxy_protocol`。
3. `xray_reality_xver` 是否为 `1`。
4. Nginx 证书是否匹配 Reality 的 `server_name`。
5. 本地域名是否被设置为直连，避免节点访问自身形成代理循环。
