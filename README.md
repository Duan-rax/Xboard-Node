# xboard-node

Node backend for [Xboard](https://github.com/cedar2025/Xboard). Supports `sing-box` / `xray-core` dual kernels.

> **Disclaimer**: This project is for educational and learning purposes only.

## Features

- Protocols: V2Ray family, Trojan, Shadowsocks, Hysteria2, TUIC, AnyTLS
- Sync: WebSocket push + REST polling dual channel
- User controls: speed limit, device limit, alive-IP tracking, hot update
- Deploy modes: node mode, machine mode, standalone mode
- Multi-instance: single process binding multiple panels / nodes

## Changes in this fork

This fork keeps the upstream node behavior and adds an explicit Xray Reality local-fallback path.

### Xray Reality local fallback

Two optional per-instance kernel fields are available:

```yaml
kernel:
  type: xray
  xray_reality_dest_override: "127.0.0.1:8001"
  xray_reality_xver: 1
```

- `xray_reality_dest_override` changes the Reality destination only in the node's generated runtime configuration. The panel-side `dest` and `server_name` remain the public hostname used by subscriptions.
- `xray_reality_xver` controls the Reality fallback PROXY protocol version. With `1`, the local Nginx listener must use `proxy_protocol`.
- This makes it possible to terminate fallback traffic on a local Nginx site without changing the public SNI or adding a host-file workaround.
- Both fields are optional. Deployments that do not use a local fallback keep the upstream behavior.

### Fork Release and installer

- Linux `amd64` and `arm64` binaries are published in this repository's `dev` Release.
- The installer downloads from `Duan-rax/Xboard-Node` by default.
- The deployment guides verify the downloaded candidate, installed binary, and running process separately so that an old or upstream binary is not mistaken for the fork build.

### Deployment guides

- [Fresh Linux deployment](docs/fresh-deployment.md): install a new Node or Machine and optionally configure Reality with a local Nginx fallback.
- [Existing Linux deployment upgrade](docs/existing-deployment-upgrade.md): safely replace an existing binary, preserve configuration, verify the running process, and roll back if necessary.

## Install

### Docker

```bash
docker run -d --restart=always --network=host \
  -e apiHost=https://panel.com -e apiKey=TOKEN -e nodeID=1 \
  ghcr.io/cedar2025/xboard-node:latest
```

### Docker Compose

```bash
git clone -b compose --depth 1 https://github.com/cedar2025/xboard-node.git
cd xboard-node
vim config/config.yml   # set panel.url / token / node_id
docker compose up -d
```

### Installer (Linux systemd)

```bash
# Node mode
curl -fsSL https://raw.githubusercontent.com/cedar2025/xboard-node/dev/install.sh | \
  sudo bash -s -- --mode node --panel https://panel.example.com --token TOKEN --node-id 1

# Machine mode
curl -fsSL https://raw.githubusercontent.com/cedar2025/xboard-node/dev/install.sh | \
  sudo bash -s -- --mode machine --panel https://panel.example.com --token TOKEN --machine-id 1
```

## xbctl

Run `xbctl` after installation for help. Common commands:

```bash
xbctl list                          # list all instances
xbctl status                        # running status
xbctl bind add-node --panel URL --token TOKEN --node-id 1
xbctl bind add-machine --panel URL --token TOKEN --machine-id 1
xbctl bind remove-node --panel URL --node-id 1
xbctl service restart
```

## Configuration

Legacy single-panel config is fully compatible. Appending bindings auto-migrates to `instances` format. See `config.yml.example`.

## Extensions

- Custom routes: [docs-custom-routes.md](docs-custom-routes.md)
- Custom outbounds: [docs-custom-outbounds.md](docs-custom-outbounds.md)
- DNS providers (ACME DNS-01): [docs-dns-providers.md](docs-dns-providers.md)
- Fresh Linux deployment: [docs/fresh-deployment.md](docs/fresh-deployment.md)
- Existing Linux deployment upgrade: [docs/existing-deployment-upgrade.md](docs/existing-deployment-upgrade.md)

## License

MPL-2.0.
