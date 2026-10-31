package model

import "testing"

func TestAvailableEgressProxiesDeduplicatesLegacyDefault(t *testing.T) {
	node := &NodeSpec{
		EgressProxy: &EgressProxyConfig{ID: 1, Host: "legacy", Port: 1080},
		EgressProxies: []EgressProxyConfig{
			{ID: 1, Host: "available", Port: 1080},
			{ID: 2, Host: "ai", Port: 443},
		},
		FinalEgressProxyID: 2,
	}
	proxies := node.AvailableEgressProxies()
	if len(proxies) != 2 {
		t.Fatalf("available egress count = %d, want 2", len(proxies))
	}
	if proxies[0].Host != "available" {
		t.Fatalf("explicit egress list should win over legacy field: %#v", proxies)
	}
	final := node.FinalEgressProxy()
	if final == nil || final.ID != 2 {
		t.Fatalf("final egress = %#v, want id 2", final)
	}
}

func TestLegacyEgressRemainsFinalWithoutExplicitFinalID(t *testing.T) {
	node := &NodeSpec{EgressProxy: &EgressProxyConfig{ID: 9, Host: "legacy", Port: 1080}}
	final := node.FinalEgressProxy()
	if final == nil || final.ID != 9 {
		t.Fatalf("legacy final egress = %#v, want id 9", final)
	}
}
