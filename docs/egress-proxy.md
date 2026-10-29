# Xboard 管理出站

当 Xboard 节点配置包含 `egress_proxy` 时，Xboard-Node 会将其编译为
当前内核的原生出站，并让未命中既有路由的流量从该出站离开。

本功能由面板统一下发，不需要在 `/etc/xboard-node/config.yml` 填写
代理账号，也不影响用户订阅。

这是一个可选扩展字段。未包含本功能的官方 Xboard-Node 会忽略
`egress_proxy` 并继续使用原有直连/自定义出站；因此同一面板可以同时连接
官方节点和定制节点，但只能把管理出站绑定到支持本扩展的节点。

| 内核 | 生成方式 |
| --- | --- |
| sing-box | 增加 `panel-egress-<id>` 出站，设置 `route.final` |
| Xray | 增加 `panel-egress-<id>` 出站，在既有规则后追加 TCP/UDP 路由 |

面板的拦截、审计、私网保护和受信任自定义路由仍具有更高优先级。一个节点
只能有一个管理出站；重新绑定会原子地替换上一个出站。

支持 SOCKS5、HTTP/HTTPS、Shadowsocks、VLESS（含 TLS/Reality）和 Trojan。
协议参数由面板的 `config` 对象传入，节点会在重载前验证主机、端口、必要的
SS method、VLESS UUID、Trojan 密码及生成标签冲突。

