package xray

import (
	"strings"

	"github.com/cedar2025/xboard-node/internal/model"
)

func egressProxyToXray(proxy *model.EgressProxyConfig) M {
	config := proxy.Config
	server := M{"address": proxy.Host, "port": proxy.Port}
	withCredentials := func() {
		if proxy.Username != "" || proxy.Password != "" {
			server["users"] = []M{{"user": proxy.Username, "pass": proxy.Password}}
		}
	}
	result := M{"tag": model.EgressProxyTag(proxy.ID)}
	switch strings.ToLower(proxy.Protocol) {
	case "socks5":
		withCredentials()
		result["protocol"] = "socks"
		result["settings"] = M{"servers": []M{server}}
	case "http":
		withCredentials()
		result["protocol"] = "http"
		result["settings"] = M{"servers": []M{server}}
	case "shadowsocks":
		server["method"] = config["method"]
		server["password"] = proxy.Password
		result["protocol"] = "shadowsocks"
		result["settings"] = M{"servers": []M{server}}
	case "vless":
		user := M{"id": config["uuid"], "encryption": "none"}
		if flow := egressStringValue(config, "flow"); flow != "" {
			user["flow"] = flow
		}
		server["users"] = []M{user}
		result["protocol"] = "vless"
		result["settings"] = M{"vnext": []M{server}}
	case "trojan":
		server["password"] = proxy.Password
		result["protocol"] = "trojan"
		result["settings"] = M{"servers": []M{server}}
	}
	if stream := egressProxyStreamSettings(proxy); stream != nil {
		result["streamSettings"] = stream
	}
	return result
}

func egressProxyStreamSettings(proxy *model.EgressProxyConfig) M {
	if !proxy.TLSEnabled && proxy.Config["reality"] == nil {
		return nil
	}
	network := egressStringValue(proxy.Config, "network")
	if network == "" {
		network = "tcp"
	}
	if reality, ok := proxy.Config["reality"].(map[string]any); ok {
		fingerprint := egressStringValue(reality, "fingerprint")
		if fingerprint == "" {
			fingerprint = egressStringValue(proxy.Config, "fingerprint")
		}
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		settings := M{
			"show":        false,
			"password":    reality["public_key"],
			"shortId":     reality["short_id"],
			"fingerprint": fingerprint,
		}
		name := proxy.TLSServerName
		if name == "" {
			name = egressStringValue(proxy.Config, "server_name")
		}
		if name != "" {
			settings["serverName"] = name
		}
		return M{"network": network, "security": "reality", "realitySettings": settings}
	}
	settings := M{"allowInsecure": proxy.AllowInsecure}
	name := proxy.TLSServerName
	if name == "" {
		name = egressStringValue(proxy.Config, "server_name")
	}
	if name != "" {
		settings["serverName"] = name
	}
	return M{"network": network, "security": "tls", "tlsSettings": settings}
}

func hasPanelDNSRoute(rules []model.RouteRule) bool {
	for _, rule := range rules {
		if !rule.Disabled && strings.EqualFold(strings.TrimSpace(rule.Action), "dns") {
			return true
		}
	}
	return false
}

func egressStringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func applyEgressProxyRoute(cfg M, proxy *model.EgressProxyConfig) {
	if proxy == nil {
		return
	}
	routing, ok := cfg["routing"].(M)
	if !ok {
		routing = M{}
		cfg["routing"] = routing
	}
	rules, _ := routing["rules"].([]M)
	// This is deliberately last: safety and panel rules retain their priority and
	// only traffic not matched by them uses the residential/provider egress.
	rules = append(rules, M{"type": "field", "network": "tcp,udp", "outboundTag": model.EgressProxyTag(proxy.ID)})
	routing["rules"] = rules
}

// outboundConfigToXray converts a structured OutboundConfig (from the panel)
// into an Xray outbound object. Xray uses a nested layout where protocol-
// specific fields go inside a "settings" key, and chain proxying uses "proxySettings".
func outboundConfigToXray(oc model.OutboundConfig) M {
	m := M{
		"protocol": oc.Protocol,
		"tag":      oc.Tag,
	}
	if len(oc.Settings) > 0 {
		m["settings"] = oc.Settings
	}
	if oc.ProxyTag != "" {
		m["proxySettings"] = M{"tag": oc.ProxyTag}
	}
	return m
}

func mergeRouteList(a, b []map[string]any) []map[string]any {
	res := make([]map[string]any, 0, len(a)+len(b))
	res = append(res, a...)
	res = append(res, b...)
	return res
}
