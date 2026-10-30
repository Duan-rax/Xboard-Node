package panel

import "testing"

func TestDecodeWeakRawIncludesEgressProxy(t *testing.T) {
	raw := map[string]interface{}{
		"protocol": "anytls",
		"egress_proxy": map[string]interface{}{
			"id":       1,
			"name":     "Legend SG",
			"protocol": "shadowsocks",
			"host":     "217.116.172.174",
			"port":     50828,
			"password": "secret",
			"config": map[string]interface{}{
				"method": "aes-256-gcm",
			},
		},
	}

	var config NodeConfig
	if err := decodeWeakRaw(raw, &config); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if config.EgressProxy == nil {
		t.Fatal("egress_proxy was not decoded")
	}
	if config.EgressProxy.Host != "217.116.172.174" || config.EgressProxy.Port != 50828 {
		t.Fatalf("unexpected egress proxy: %#v", config.EgressProxy)
	}
	if got := config.EgressProxy.Config["method"]; got != "aes-256-gcm" {
		t.Fatalf("unexpected method: %#v", got)
	}
}

func TestDecodeWeakRawIncludesStructuredRoutes(t *testing.T) {
	raw := map[string]interface{}{
		"protocol": "vless",
		"routes": []interface{}{
			map[string]interface{}{
				"id":           9,
				"remarks":      "structured",
				"match":        []interface{}{},
				"criteria":     map[string]interface{}{"domain_suffixes": []interface{}{"example.com"}},
				"match_mode":   "all",
				"priority":     15,
				"disabled":     true,
				"action":       "proxy",
				"action_value": "warp",
			},
		},
	}

	var config NodeConfig
	if err := decodeWeakRaw(raw, &config); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if len(config.Routes) != 1 || config.Routes[0].Criteria == nil {
		t.Fatalf("structured route was not decoded: %#v", config.Routes)
	}
	route := config.Routes[0]
	if route.MatchMode != "all" || route.Priority != 15 || !route.Disabled || route.ActionValue != "warp" {
		t.Fatalf("unexpected route metadata: %#v", route)
	}
	if got := route.Criteria.DomainSuffixes[0]; got != "example.com" {
		t.Fatalf("unexpected route criteria: %#v", route.Criteria)
	}
}

func TestDecodeWeakRawIncludesConfigFingerprint(t *testing.T) {
	raw := map[string]interface{}{
		"protocol":        "vless",
		"server_port":     443,
		"config_revision": int64(108),
		"config_hash":     "abc123",
	}

	var config NodeConfig
	if err := decodeWeakRaw(raw, &config); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if config.ConfigRevision != 108 || config.ConfigHash != "abc123" {
		t.Fatalf("unexpected config fingerprint: %#v", config)
	}
}
