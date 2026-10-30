package model

import (
	"strings"
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
)

func TestValidateNodeSpec_EgressProtocolFields(t *testing.T) {
	tests := []struct {
		name    string
		proxy   *EgressProxyConfig
		wantErr string
	}{
		{
			name: "shadowsocks password",
			proxy: &EgressProxyConfig{ID: 1, Protocol: "shadowsocks", Host: "proxy.test", Port: 8388, Config: map[string]any{"method": "aes-256-gcm"}},
			wantErr: "shadowsocks requires a password",
		},
		{
			name: "vless UUID",
			proxy: &EgressProxyConfig{ID: 1, Protocol: "vless", Host: "proxy.test", Port: 443, Config: map[string]any{"uuid": "not-a-uuid"}},
			wantErr: "standard 36-character UUID",
		},
		{
			name: "reality even short id",
			proxy: &EgressProxyConfig{ID: 1, Protocol: "vless", Host: "proxy.test", Port: 443, TLSServerName: "cdn.test", Config: map[string]any{
				"uuid": "6f0c30dc-5e98-4eb1-a52a-042ea7861e08",
				"reality": map[string]any{"public_key": "key", "short_id": "abc"},
			}},
			wantErr: "even-length hexadecimal",
		},
		{
			name: "valid reality",
			proxy: &EgressProxyConfig{ID: 1, Protocol: "vless", Host: "proxy.test", Port: 443, TLSServerName: "cdn.test", Config: map[string]any{
				"uuid": "6f0c30dc-5e98-4eb1-a52a-042ea7861e08",
				"reality": map[string]any{"public_key": "key", "short_id": "abcd"},
			}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNodeSpec(&NodeSpec{EgressProxy: tc.proxy}, config.KernelConfig{Type: "singbox"})
			if tc.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidatePanelRouteRules_RejectsUnknownNodeOutbound(t *testing.T) {
	err := ValidatePanelRouteRules([]RouteRule{{
		Criteria:    &RouteMatch{DomainSuffixes: []string{"example.com"}},
		Action:      "proxy",
		ActionValue: "panel-egress-99",
	}}, "xray", map[string]struct{}{"direct": {}, "block": {}})
	if err == nil || !strings.Contains(err.Error(), `unknown outbound "panel-egress-99"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePanelRouteRules_AllowsDNSAction(t *testing.T) {
	err := ValidatePanelRouteRules([]RouteRule{{
		Criteria: &RouteMatch{Ports: []string{"53"}, Networks: []string{"udp"}},
		Action:   "dns",
	}}, "singbox", map[string]struct{}{"direct": {}, "block": {}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
