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
