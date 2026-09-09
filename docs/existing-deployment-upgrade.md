# Xboard-Node 现有部署升级

本文适用于已经通过 systemd 运行 Xboard-Node 的服务器。升级只替换后端与 `xbctl`，保留现有配置、实例目录和凭据。

本文不包含流量采集、访问审计、RiskAudit 插件或 Clash API。

## 1. 先确认当前安装

```bash
sudo -i

systemctl cat xboard-node
systemctl show xboard-node -p ExecStart -p EnvironmentFiles --no-pager
PID=$(pidof xboard-node)
readlink -f "/proc/$PID/exe"
sha256sum "/proc/$PID/exe" /usr/local/bin/xboard-node
```

标准安装的 `ExecStart` 应指向：

```text
/usr/local/bin/xboard-node -c /etc/xboard-node/config.yml
```

如果服务使用其他路径，必须以实际 `ExecStart` 为准，不能直接假设替换 `/usr/local/bin/xboard-node` 会生效。

## 2. 备份

安装器会自动备份，但升级前建议再保留一份独立副本：

```bash
STAMP=$(date +%Y%m%d-%H%M%S)
BACKUP=/root/xboard-node-backup-$STAMP

install -d -m 0700 "$BACKUP"
cp -a /etc/xboard-node "$BACKUP/"
cp -a /usr/local/bin/xboard-node "$BACKUP/"
[ -x /usr/local/bin/xbctl ] && cp -a /usr/local/bin/xbctl "$BACKUP/" || true
systemctl cat xboard-node > "$BACKUP/xboard-node.service.txt"

echo "Backup: $BACKUP"
```

## 3. 下载并验证候选文件

```bash
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

UPGRADE_DIR=/root/xboard-node-upgrade
install -d -m 0700 "$UPGRADE_DIR"
cd "$UPGRADE_DIR"

RELEASE_BASE=https://github.com/Duan-rax/Xboard-Node/releases/download/dev

curl -fL --retry 3 -o install.sh   https://raw.githubusercontent.com/Duan-rax/Xboard-Node/dev/install.sh
curl -fL --retry 3 -o xboard-node.new   "$RELEASE_BASE/xboard-node-linux-${ARCH}"
curl -fL --retry 3 -o xbctl.new   "$RELEASE_BASE/xbctl-linux-${ARCH}"

chmod 700 install.sh xboard-node.new xbctl.new
./xboard-node.new -v
./xbctl.new version
sha256sum xboard-node.new xbctl.new
```

Reality 本地回落用户还应验证功能字段：

```bash
for key in xray_reality_dest_override xray_reality_xver; do
  grep -aq "$key" ./xboard-node.new     && echo "$key: present"     || { echo "$key: MISSING"; exit 1; }
done
```

任何字段显示 `MISSING` 时立即停止，不要替换当前服务。

## 4. 执行升级

必须把已经验证的候选文件通过绝对路径传给安装器：

```bash
cd /root/xboard-node-upgrade

bash ./install.sh upgrade   --binary "$PWD/xboard-node.new"   --xbctl-binary "$PWD/xbctl.new"
```

不要省略 `--binary`。省略它会让安装器自行选择下载源，无法证明安装的就是上一步验证过的文件。

升级过程会：

1. 备份当前二进制、配置、凭据和 service。
2. 安装候选二进制。
3. 重新加载 systemd 并重启服务。
4. 执行健康检查；失败时触发回滚。

## 5. 验证替换结果

```bash
PID=$(pidof xboard-node)

systemctl is-active xboard-node
systemctl show xboard-node -p ExecStart --no-pager
readlink -f "/proc/$PID/exe"

sha256sum   /root/xboard-node-upgrade/xboard-node.new   /usr/local/bin/xboard-node   "/proc/$PID/exe"

cmp -s /root/xboard-node-upgrade/xboard-node.new /usr/local/bin/xboard-node
cmp -s /usr/local/bin/xboard-node "/proc/$PID/exe"
echo "candidate, installed binary and running process are identical"
```

最后确认 Reality 字段确实存在于运行中进程：

```bash
for key in xray_reality_dest_override xray_reality_xver; do
  grep -aq "$key" "/proc/$PID/exe"     && echo "$key: present"     || { echo "$key: MISSING"; exit 1; }
done

xbctl status
journalctl -u xboard-node -n 100 --no-pager
```

三个 SHA-256 应完全相同。只比较 `/proc/$PID/exe` 与 `/usr/local/bin/xboard-node` 不足以证明它们等于刚下载的候选文件，必须把候选文件也放在同一次比较中。

## 6. 配置 Reality 本地回落

升级成功后，在目标实例的 `kernel` 下加入：

```yaml
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8001"
  xray_reality_xver: 1
```

面板 Reality 的 `dest` 和 `server_name` 保留公网域名。不要把 `127.0.0.1:8001` 写回面板。

Nginx 必须使用匹配配置：

```nginx
listen 127.0.0.1:8001 ssl proxy_protocol;
set_real_ip_from 127.0.0.1;
real_ip_header proxy_protocol;
```

检查并重启：

```bash
nginx -t
systemctl reload nginx
xbctl restart
```

直接测试 Nginx：

```bash
curl --haproxy-protocol   --resolve test.example.com:8001:127.0.0.1   -kiv https://test.example.com:8001/
```

测试完整链路：

```bash
curl --resolve test.example.com:443:127.0.0.1   -kiv https://test.example.com/
```

## 7. 为什么以前会出现“替换失败”

容易混淆的地方有三处：

1. `curl -o xboard-node ...` 只下载文件，并未安装到 `/usr/local/bin`。
2. `./xboard-node -v` 只运行下载目录中的文件，不代表 systemd 使用它。
3. 旧版 fork 安装器的默认下载源仍指向上游仓库；升级时若未显式传入 `--binary`，可能重新下载并安装不含 fork 扩展字段的上游版本。

因此本教程始终要求：

- 先验证候选文件；
- 使用 `install.sh upgrade --binary ...` 明确替换；
- 最后比较候选文件、安装文件和运行中进程三者的哈希。

## 8. 回滚

先查看安装器自动生成的备份：

```bash
ls -lah /etc/xboard-node/backups
```

如果升级后服务异常，优先查看安装器日志和自动回滚结果：

```bash
systemctl status xboard-node --no-pager
journalctl -u xboard-node -n 200 --no-pager
```

需要手动恢复时，使用第 2 节生成的确切备份目录：

```bash
systemctl stop xboard-node
install -m 0755 /root/xboard-node-backup-<时间戳>/xboard-node /usr/local/bin/xboard-node
systemctl start xboard-node
systemctl is-active xboard-node
```

确认服务恢复后再处理配置。不要删除备份目录，直到节点稳定运行。
