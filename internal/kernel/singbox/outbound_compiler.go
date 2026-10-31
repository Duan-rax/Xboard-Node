package singbox

import (
	"net"
	"strconv"
	"strings"

	"github.com/cedar2025/xboard-node/internal/model"
)

func hasPanelDNSRoute(rules []model.RouteRule) bool {
	for _, rule := range rules {
		if !rule.Disabled && strings.EqualFold(strings.TrimSpace(rule.Action), "dns") {
			return true
		}
	}
	return false
}

func egressProxyToSingbox(proxy *model.EgressProxyConfig) M {
	tag := model.EgressProxyTag(proxy.ID)
	config := proxy.Config
	base := M{"tag": tag, "server": proxy.Host, "server_port": proxy.Port}
	userPassword := func(m M) {
		if proxy.Username != "" {
			m["username"] = proxy.Username
		}
		if proxy.Password != "" {
			m["password"] = proxy.Password
		}
	}
	tls := egressProxyTLS(proxy)
	switch strings.ToLower(proxy.Protocol) {
	case "socks5":
		base["type"] = "socks"
		base["version"] = "5"
		userPassword(base)
	case "http":
		base["type"] = "http"
		userPassword(base)
		if tls != nil {
			base["tls"] = tls
		}
	case "shadowsocks":
		base["type"] = "shadowsocks"
		base["method"] = config["method"]
		base["password"] = proxy.Password
	case "vless":
		base["type"] = "vless"
		base["uuid"] = config["uuid"]
		if flow := egressStringValue(config, "flow"); flow != "" {
			base["flow"] = flow
		}
		if tls != nil {
			base["tls"] = tls
		}
	case "trojan":
		base["type"] = "trojan"
		base["password"] = proxy.Password
		if tls != nil {
			base["tls"] = tls
		}
	}
	return base
}

func egressProxyTLS(proxy *model.EgressProxyConfig) M {
	if !proxy.TLSEnabled && proxy.Config["reality"] == nil {
		return nil
	}
	tls := M{"enabled": true, "insecure": proxy.AllowInsecure}
	serverName := proxy.TLSServerName
	if serverName == "" {
		serverName = egressStringValue(proxy.Config, "server_name")
	}
	if serverName != "" {
		tls["server_name"] = serverName
	}
	if reality, ok := proxy.Config["reality"].(map[string]any); ok {
		tls["reality"] = M{"enabled": true, "public_key": reality["public_key"], "short_id": reality["short_id"]}
		fingerprint := egressStringValue(reality, "fingerprint")
		if fingerprint == "" {
			fingerprint = egressStringValue(proxy.Config, "fingerprint")
		}
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		tls["utls"] = M{"enabled": true, "fingerprint": fingerprint}
	}
	return tls
}

func egressStringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func applyEgressProxyRoute(cfg M, proxy *model.EgressProxyConfig) {
	if proxy == nil {
		return
	}
	route, ok := cfg["route"].(M)
	if !ok {
		route = M{}
		cfg["route"] = route
	}
	rules, _ := route["rules"].([]M)
	// Keep the configured final outbound intact. The egress is a catch-all
	// rule after custom and panel rules so hot reload updates it reliably.
	rules = append(rules, M{"action": "route", "outbound": model.EgressProxyTag(proxy.ID)})
	route["rules"] = rules
}

// outboundConfigToSingbox converts a structured OutboundConfig (from the panel)
// into a sing-box outbound object. sing-box uses a flat layout where all
// protocol-specific fields sit at the top level alongside "type" and "tag".
func outboundConfigToSingbox(oc model.OutboundConfig) M {
	m := M{
		"type": oc.Protocol,
		"tag":  oc.Tag,
	}

	// Transform common protocol keys to sing-box native format
	// WireGuard: secret_key -> private_key; peers: endpoint -> address+port
	if oc.Protocol == "wireguard" {
		if sk, ok := oc.Settings["secret_key"]; ok {
			m["private_key"] = sk
		}
		if peers, ok := oc.Settings["peers"].([]any); ok {
			var wgPeers []M
			for _, p := range peers {
				if peerMap, ok := p.(map[string]any); ok {
					newPeer := M{}
					for k, v := range peerMap {
						if k == "endpoint" {
							if ep, ok := v.(string); ok {
								host, portStr, err := net.SplitHostPort(ep)
								if err == nil {
									newPeer["address"] = host
									port, _ := strconv.Atoi(portStr)
									newPeer["port"] = port
								} else {
									newPeer["address"] = ep
								}
							}
						} else {
							newPeer[k] = v
						}
					}
					wgPeers = append(wgPeers, newPeer)
				}
			}
			m["peers"] = wgPeers
		}

		// Copy any other top-level settings not handled above
		for k, v := range oc.Settings {
			if k != "secret_key" && k != "peers" {
				m[k] = v
			}
		}
	} else {
		for k, v := range oc.Settings {
			m[k] = v
		}
	}

	if oc.ProxyTag != "" {
		m["proxy_tag"] = oc.ProxyTag
	}
	return m
}

func mergeRouteList(a, b []map[string]any) []map[string]any {
	res := make([]map[string]any, 0, len(a)+len(b))
	res = append(res, a...)
	res = append(res, b...)
	return res
}
