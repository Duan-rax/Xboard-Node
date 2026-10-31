package probe

import (
	"testing"

	"github.com/cedar2025/xboard-node/internal/model"
)

func TestValidateTargetRejectsUnsupportedScheme(t *testing.T) {
	if err := validateTarget("ftp://example.com/file"); err == nil {
		t.Fatal("expected unsupported scheme to be rejected")
	}
	if err := validateTarget("https://api.ipify.org?format=json"); err != nil {
		t.Fatalf("valid target rejected: %v", err)
	}
}

func TestProbeConfigMapsProtocolSpecificFields(t *testing.T) {
	cfg := probeConfig(&model.EgressProxyConfig{
		Protocol: "vless",
		Host:     "203.0.113.10",
		Port:     443,
		TLSEnabled: true,
		TLSServerName: "example.com",
		Config: map[string]any{
			"uuid": "00000000-0000-0000-0000-000000000001",
			"flow": "xtls-rprx-vision",
		},
	}, 12345)
	outbounds := cfg["outbounds"].([]any)
	outbound := outbounds[0].(map[string]any)
	if outbound["type"] != "vless" || outbound["uuid"] == "" || outbound["flow"] != "xtls-rprx-vision" {
		t.Fatalf("unexpected VLESS outbound: %#v", outbound)
	}
	tlsOptions, ok := outbound["tls"].(map[string]any)
	if !ok || tlsOptions["server_name"] != "example.com" {
		t.Fatalf("unexpected TLS options: %#v", outbound["tls"])
	}
}

func TestParseExitIP(t *testing.T) {
	if got := parseExitIP([]byte(`{"ip":"203.0.113.8"}`)); got != "203.0.113.8" {
		t.Fatalf("exit IP = %q", got)
	}
	if got := parseExitIP([]byte(`{"ip":"not-an-ip"}`)); got != "" {
		t.Fatalf("invalid exit IP = %q", got)
	}
}
