# Xray Reality runtime overrides

This fork separates the Reality address used by Xboard for client subscription
generation from the address used by the node process at runtime.

## Why this exists

Some Xboard panel versions expose only one Reality `dest` value. If you put a
local listener such as `127.0.0.1:8001` in that field, generated client
subscriptions may use an invalid SNI. If you put the public SNI there instead,
the node cannot send PROXY protocol to a local Nginx/Caddy listener.

The local overrides solve this without changing `/etc/hosts`:

```yaml
instances:
  - id: panel-example-machine-10
    kernel:
      type: xray
      xray_reality_dest_override: "127.0.0.1:8001"
      xray_reality_xver: 1
```

Keep the panel Reality destination public, for example:

```json
{
  "dest": "sg.keado.net:443"
}
```

When the panel does not provide a separate Reality server name, the node
derives `serverNames` from that public destination before applying the local
override. The generated Xray Reality settings become:

```json
{
  "dest": "127.0.0.1:8001",
  "xver": 1,
  "serverNames": ["sg.keado.net"]
}
```

## xver requirements

`xray_reality_xver: 1` sends text PROXY protocol v1 to the runtime `dest`.
Only use it with a listener you control and configure that listener to accept
PROXY protocol. A minimal Nginx example is:

```nginx
server {
    listen 127.0.0.1:8001 proxy_protocol;
    set_real_ip_from 127.0.0.1;
    real_ip_header proxy_protocol;
}
```

Do not set `xver` to `1` or `2` if the runtime destination is a public website
or any service that does not support PROXY protocol; its TLS handshake will
fail. Use `0` or omit the setting in that case.

## Build and deploy

1. Build the `dev` branch for the server architecture. The repository Makefile
   produces `xboard-node-linux-amd64` and `xboard-node-linux-arm64`.
2. Back up the installed binary before replacing it.
3. Add the per-instance `kernel` settings above to `/etc/xboard-node/config.yml`.
4. Ensure the local Nginx/Caddy listener is started before restarting the node.
5. Restart with `xbctl service restart` and inspect `journalctl -u xboard-node -n 100`.

The override is Xray-only. It has no effect on sing-box instances.
