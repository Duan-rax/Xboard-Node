package singbox

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/kernel"
	"github.com/cedar2025/xboard-node/internal/model"
	"github.com/cedar2025/xboard-node/internal/panel"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singJSON "github.com/sagernet/sing/common/json"
)

var testUsersPanel = []panel.User{
	{ID: 1, UUID: "aaaaaaaa-1111-2222-3333-444444444444", SpeedLimit: 0, DeviceLimit: 0},
	{ID: 2, UUID: "bbbbbbbb-5555-6666-7777-888888888888", SpeedLimit: 3, DeviceLimit: 2},
}

// --- Shadowsocks ---

var testUsers = model.UserSpecsFromPanel(testUsersPanel)

func testNodeSpec(nc *panel.NodeConfig) *model.NodeSpec { return model.NodeSpecFromPanel(nc) }

func testRouteRules(r []panel.RouteRule) []model.RouteRule {
	if r == nil {
		return nil
	}
	return model.NodeSpecFromPanel(&panel.NodeConfig{Routes: r}).Routes
}

func TestBuildInbound_Shadowsocks(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "shadowsocks",
		ServerPort: 111,
		Cipher:     "aes-128-gcm",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	assertMapValue(t, inbound, "type", "shadowsocks")
	assertMapValue(t, inbound, "method", "aes-128-gcm")
	assertMapValue(t, inbound, "listen_port", 111)

	users := inbound["users"].([]M)
	if len(users) != 2 {
		t.Fatalf("users: got %d, want 2", len(users))
	}
	assertMapValue(t, users[0], "name", "aaaaaaaa-1111-2222-3333-444444444444")
	assertMapValue(t, users[0], "password", "aaaaaaaa-1111-2222-3333-444444444444")
}

func TestBuildInbound_Shadowsocks2022(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "shadowsocks",
		ServerPort: 222,
		Cipher:     "2022-blake3-aes-128-gcm",
		ServerKey:  "base64serverkey==",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	assertMapValue(t, inbound, "method", "2022-blake3-aes-128-gcm")
	assertMapValue(t, inbound, "password", "base64serverkey==")
}

// --- VMess ---

func TestBuildInbound_VMess(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 443,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	assertMapValue(t, inbound, "type", "vmess")
	assertMapValue(t, inbound, "tag", "vmess-in")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "uuid", "aaaaaaaa-1111-2222-3333-444444444444")
	assertMapValue(t, users[0], "alterId", 0)
}

func TestBuildInbound_VMess_WithTLS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 443,
		TLS:        1,
		ServerName: "example.com",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)
	assertMapValue(t, tls, "server_name", "example.com")
	assertMapValue(t, tls, "certificate", []string{"CERT"})
	assertMapValue(t, tls, "key", []string{"KEY"})
}

// Panel tls=1 with cert_mode none (no paths): inbound must be plain — TLS terminated at nginx/CDN.
func TestBuildInbound_VMess_TLS_PanelOn_NoCertFiles(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 8888,
		Network:    "ws",
		TLS:        1,
		ServerName: "example.com",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	if _, exists := inbound["tls"]; exists {
		t.Fatal("expected no tls block when cert/key paths are empty (nginx offload)")
	}
}

func TestBuildInbound_VMess_NoTLS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 80,
		TLS:        0,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	if _, exists := inbound["tls"]; exists {
		t.Error("should not have TLS when tls=0")
	}
}

func TestBuildInbound_VMess_WithWebSocket(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 80,
		Network:    "ws",
		NetworkSettings: map[string]interface{}{
			"path": "/ws",
			"headers": map[string]interface{}{
				"Host": "example.com",
			},
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	transport := inbound["transport"].(M)
	assertMapValue(t, transport, "type", "ws")
	assertMapValue(t, transport, "path", "/ws")
}

func TestBuildInbound_VMess_WithGRPC(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 443,
		Network:    "grpc",
		NetworkSettings: map[string]interface{}{
			"serviceName": "mygrpc",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	transport := inbound["transport"].(M)
	assertMapValue(t, transport, "type", "grpc")
	assertMapValue(t, transport, "service_name", "mygrpc")
}

func TestBuildInbound_VMess_WithGRPCSnakeCaseServiceName(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 443,
		Network:    "grpc",
		NetworkSettings: map[string]interface{}{
			"service_name": "mygrpc-snake",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	transport := inbound["transport"].(M)
	assertMapValue(t, transport, "type", "grpc")
	assertMapValue(t, transport, "service_name", "mygrpc-snake")
}

func TestBuildInbound_VMess_WithH2(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 443,
		Network:    "h2",
		NetworkSettings: map[string]interface{}{
			"path": "/h2path",
			"host": "example.com",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	transport := inbound["transport"].(M)
	assertMapValue(t, transport, "type", "http")
	assertMapValue(t, transport, "path", "/h2path")
}

func TestBuildInbound_VMess_WithHTTPUpgrade(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vmess",
		ServerPort: 443,
		Network:    "httpupgrade",
		NetworkSettings: map[string]interface{}{
			"path": "/upgrade",
			"host": "example.com",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	transport := inbound["transport"].(M)
	assertMapValue(t, transport, "type", "httpupgrade")
	assertMapValue(t, transport, "path", "/upgrade")
	assertMapValue(t, transport, "host", "example.com")
}

// --- VLESS ---

func TestBuildInbound_VLESS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vless",
		ServerPort: 443,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	assertMapValue(t, inbound, "type", "vless")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "uuid", "aaaaaaaa-1111-2222-3333-444444444444")
}

func TestBuildInbound_VLESS_WithFlow(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vless",
		ServerPort: 443,
		Flow:       "xtls-rprx-vision",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "flow", "xtls-rprx-vision")
}

func TestBuildInbound_VLESS_Reality(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vless",
		ServerPort: 443,
		TLS:        2,
		TLSSettings: map[string]interface{}{
			"private_key": "test-private-key",
			"short_id":    "abc123",
			"dest":        "www.example.com:443",
			"server_name": "www.example.com",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)

	reality := tls["reality"].(M)
	assertMapValue(t, reality, "enabled", true)
	assertMapValue(t, reality, "private_key", "test-private-key")

	handshake := reality["handshake"].(M)
	assertMapValue(t, handshake, "server", "www.example.com")
	assertMapValue(t, handshake, "server_port", 443)
}

func TestBuildInbound_VLESS_Reality_ShortIDArray(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "vless",
		ServerPort: 443,
		TLS:        2,
		TLSSettings: map[string]interface{}{
			"private_key": "pk",
			"short_id":    []interface{}{"id1", "id2"},
			"dest":        "example.com",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	reality := inbound["tls"].(M)["reality"].(M)
	ids := reality["short_id"].([]string)
	if len(ids) != 2 {
		t.Errorf("short_id: got %d entries, want 2", len(ids))
	}
}

// --- Trojan ---

func TestBuildInbound_Trojan_Reality(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "trojan",
		ServerPort: 443,
		TLS:        2,
		TLSSettings: map[string]interface{}{
			"private_key": "test-private-key",
			"dest":        "gateway.example.com:8443",
			"server_name": "cdn.example.com",
		},
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)
	assertMapValue(t, tls, "server_name", "cdn.example.com")

	reality := tls["reality"].(M)
	assertMapValue(t, reality, "enabled", true)
	assertMapValue(t, reality, "private_key", "test-private-key")

	handshake := reality["handshake"].(M)
	assertMapValue(t, handshake, "server", "gateway.example.com")
	assertMapValue(t, handshake, "server_port", 8443)
}

func TestBuildInbound_Trojan_NoTLS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "trojan",
		ServerPort: 80,
		TLS:        0,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	// Without certificate paths there is no TLS layer; sing-box trojan needs certs or Reality.
	if _, exists := inbound["tls"]; exists {
		t.Fatal("expected no tls when cert paths are empty and tls=0")
	}
}

func TestBuildInbound_Trojan_WithTLS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "trojan",
		ServerPort: 443,
		TLS:        1,
		ServerName: "example.com",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)
}

// --- Hysteria ---

func TestBuildInbound_Hysteria2(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "hysteria",
		ServerPort: 444,
		Version:    2,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, inbound, "type", "hysteria2")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "password", "aaaaaaaa-1111-2222-3333-444444444444")

	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)
}

func TestBuildInbound_Hysteria2_WithObfs(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:     "hysteria",
		ServerPort:   444,
		Version:      2,
		Obfs:         "salamander",
		ObfsPassword: "secret",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	obfs := inbound["obfs"].(M)
	assertMapValue(t, obfs, "type", "salamander")
	assertMapValue(t, obfs, "password", "secret")
}

func TestBuildInbound_Hysteria1(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "hysteria",
		ServerPort: 444,
		Version:    0,
		UpMbps:     100,
		DownMbps:   200,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, inbound, "type", "hysteria")
	assertMapValue(t, inbound, "up_mbps", 100)
	assertMapValue(t, inbound, "down_mbps", 200)

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "auth_str", "aaaaaaaa-1111-2222-3333-444444444444")
}

// --- TUIC ---

func TestBuildInbound_TUIC(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:          "tuic",
		ServerPort:        555,
		CongestionControl: "bbr",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, inbound, "type", "tuic")
	assertMapValue(t, inbound, "congestion_control", "bbr")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "uuid", "aaaaaaaa-1111-2222-3333-444444444444")
	assertMapValue(t, users[0], "password", "aaaaaaaa-1111-2222-3333-444444444444")
}

// --- AnyTLS ---

func TestBuildInbound_AnyTLS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:      "anytls",
		ServerPort:    443,
		PaddingScheme: "stop=8\n0=30-30",
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, inbound, "type", "anytls")
	assertMapValue(t, inbound, "padding_scheme", "stop=8\n0=30-30")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "password", "aaaaaaaa-1111-2222-3333-444444444444")
}

func TestBuildInbound_AnyTLS_NoPaddingScheme(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "anytls",
		ServerPort: 443,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	if _, exists := inbound["padding_scheme"]; exists {
		t.Error("should not have padding_scheme when empty")
	}
}

// --- Naive ---

func TestBuildInbound_Naive(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "naive",
		ServerPort: 443,
		TLS:        1,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, inbound, "type", "naive")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "username", "1")
	assertMapValue(t, users[0], "password", "aaaaaaaa-1111-2222-3333-444444444444")

	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)
}

// --- Socks ---

func TestBuildInbound_Socks(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "socks",
		ServerPort: 1080,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	assertMapValue(t, inbound, "type", "socks")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "username", "aaaaaaaa-1111-2222-3333-444444444444")
	assertMapValue(t, users[0], "password", "aaaaaaaa-1111-2222-3333-444444444444")

	if _, exists := inbound["tls"]; exists {
		t.Error("socks should not have TLS")
	}
}

// --- HTTP ---

func TestBuildInbound_HTTP(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "http",
		ServerPort: 8080,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	assertMapValue(t, inbound, "type", "http")

	users := inbound["users"].([]M)
	assertMapValue(t, users[0], "username", "aaaaaaaa-1111-2222-3333-444444444444")
}

func TestBuildInbound_HTTP_WithTLS(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "http",
		ServerPort: 443,
		TLS:        1,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	tls := inbound["tls"].(M)
	assertMapValue(t, tls, "enabled", true)
}

// --- Unknown protocol ---

func TestBuildInbound_Unknown(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol:   "unknown-proto",
		ServerPort: 999,
	}
	inbound := buildInbound(testNodeSpec(nc), testUsers, kernel.TLSCert{})
	if inbound != nil {
		t.Errorf("unknown protocol should return nil, got %v", inbound)
	}
}

// --- buildConfig ---

func TestBuildConfig(t *testing.T) {
	kcfg := config.KernelConfig{LogLevel: "info"}
	nc := &panel.NodeConfig{
		Protocol:   "shadowsocks",
		ServerPort: 111,
		Cipher:     "aes-128-gcm",
	}
	cfg := buildConfig(kcfg, testNodeSpec(nc), testUsers, kernel.TLSCert{})

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty JSON output")
	}

	logCfg := cfg["log"].(M)
	assertMapValue(t, logCfg, "level", "info")
	assertMapValue(t, logCfg, "timestamp", true)

	outbounds := cfg["outbounds"].([]M)
	if len(outbounds) != 2 {
		t.Errorf("outbounds: got %d, want 2", len(outbounds))
	}
	assertMapValue(t, outbounds[0], "type", "direct")
	assertMapValue(t, outbounds[1], "type", "block")

	inbounds := cfg["inbounds"].([]M)
	if len(inbounds) != 1 {
		t.Fatalf("inbounds: got %d, want 1", len(inbounds))
	}
	assertMapValue(t, inbounds[0], "type", "shadowsocks")
}

func TestBuildConfig_OutboundPriority(t *testing.T) {
	kcfg := config.KernelConfig{
		LogLevel: "info",
		CustomOutbound: []map[string]any{
			{"tag": "block", "type": "dns"}, // Local static override
		},
	}
	nc := &panel.NodeConfig{
		Protocol:   "shadowsocks",
		ServerPort: 111,
		Cipher:     "aes-128-gcm",
		CustomOutbounds: []panel.OutboundConfig{
			{Tag: "direct", Protocol: "socks", Settings: map[string]any{"address": "1.2.3.4"}}, // Panel override
		},
	}

	cfg := buildConfig(kcfg, testNodeSpec(nc), testUsers, kernel.TLSCert{})
	outbounds := cfg["outbounds"].([]M)

	// We have 2 overrides in input, so we should have exactly 2 outbounds total
	// (no auto-generated defaults for these tags)
	if len(outbounds) != 2 {
		t.Errorf("outbounds: got %d, want 2", len(outbounds))
	}

	foundDirect := false
	foundBlock := false

	for _, o := range outbounds {
		tag := o["tag"].(string)
		if tag == "direct" {
			foundDirect = true
			if o["type"] != "socks" {
				t.Errorf("expected 'direct' outbound to be type 'socks' (panel priority), got %v", o["type"])
			}
		}
		if tag == "block" {
			foundBlock = true
			if o["type"] != "dns" {
				t.Errorf("expected 'block' outbound to be type 'dns' (static config priority), got %v", o["type"])
			}
		}
	}

	if !foundDirect || !foundBlock {
		t.Errorf("missing outbounds: direct=%v, block=%v", foundDirect, foundBlock)
	}
}

func TestBuildConfig_EgressProxyAddsCatchAllRule(t *testing.T) {
	nc := &panel.NodeConfig{
		Protocol: "shadowsocks", ServerPort: 8388, Cipher: "aes-128-gcm",
		EgressProxy: &panel.EgressProxyConfig{ID: 42, Protocol: "socks5", Host: "198.51.100.8", Port: 1080, Username: "u", Password: "p"},
	}
	cfg := buildConfig(config.KernelConfig{Type: "singbox", LogLevel: "warn"}, testNodeSpec(nc), testUsers, kernel.TLSCert{})
	route := cfg["route"].(M)
	if route["final"] != "direct" {
		t.Fatalf("route final = %v, want direct", route["final"])
	}
	rules := route["rules"].([]M)
	if len(rules) == 0 || rules[len(rules)-1]["outbound"] != "panel-egress-42" {
		t.Fatalf("last route rule = %#v, want panel-egress-42", rules)
	}
	var found M
	for _, outbound := range cfg["outbounds"].([]M) {
		if outbound["tag"] == "panel-egress-42" {
			found = outbound
			break
		}
	}
	if found == nil || found["type"] != "socks" || found["server"] != "198.51.100.8" {
		t.Fatalf("unexpected egress outbound: %#v", found)
	}
}

func TestEgressProxyToSingbox_ProtocolSpecificCredentialsAndReality(t *testing.T) {
	ss := egressProxyToSingbox(&model.EgressProxyConfig{
		ID: 1, Protocol: "shadowsocks", Host: "proxy.test", Port: 8388,
		Username: "stale-user", Password: "secret", Config: map[string]any{"method": "aes-256-gcm"},
	})
	if _, exists := ss["username"]; exists {
		t.Fatalf("shadowsocks outbound contains unsupported username: %#v", ss)
	}
	assertMapValue(t, ss, "password", "secret")

	trojan := egressProxyToSingbox(&model.EgressProxyConfig{
		ID: 2, Protocol: "trojan", Host: "proxy.test", Port: 443,
		Username: "stale-user", Password: "secret",
	})
	if _, exists := trojan["username"]; exists {
		t.Fatalf("trojan outbound contains unsupported username: %#v", trojan)
	}

	vless := egressProxyToSingbox(&model.EgressProxyConfig{
		ID: 3, Protocol: "vless", Host: "proxy.test", Port: 443, TLSServerName: "cdn.test",
		Config: map[string]any{
			"uuid":    "6f0c30dc-5e98-4eb1-a52a-042ea7861e08",
			"reality": map[string]any{"public_key": "key", "short_id": "abcd"},
		},
	})
	tls := vless["tls"].(M)
	utls := tls["utls"].(M)
	assertMapValue(t, utls, "fingerprint", "chrome")
}

func TestBuildRoutes_DNSUsesHijackAction(t *testing.T) {
	route := buildRoutes([]model.RouteRule{{
		Criteria: &model.RouteMatch{Ports: []string{"53"}, Networks: []string{"udp"}},
		Action:   "dns", MatchMode: "all",
	}}, nil, nil)
	rules := route["rules"].([]M)
	last := rules[len(rules)-1]
	assertMapValue(t, last, "action", "hijack-dns")
	if _, exists := last["outbound"]; exists {
		t.Fatalf("DNS hijack rule must not reference an outbound: %#v", last)
	}
}

func TestBuildConfig_AllProtocols_ValidJSON(t *testing.T) {
	protocols := []struct {
		name string
		nc   *panel.NodeConfig
	}{
		{"shadowsocks", &panel.NodeConfig{Protocol: "shadowsocks", ServerPort: 111, Cipher: "aes-128-gcm"}},
		{"vmess", &panel.NodeConfig{Protocol: "vmess", ServerPort: 443}},
		{"vless", &panel.NodeConfig{Protocol: "vless", ServerPort: 443}},
		{"trojan", &panel.NodeConfig{Protocol: "trojan", ServerPort: 443}},
		{"hysteria2", &panel.NodeConfig{Protocol: "hysteria", ServerPort: 444, Version: 2}},
		{"hysteria1", &panel.NodeConfig{Protocol: "hysteria", ServerPort: 444, Version: 1}},
		{"tuic", &panel.NodeConfig{Protocol: "tuic", ServerPort: 555}},
		{"anytls", &panel.NodeConfig{Protocol: "anytls", ServerPort: 443}},
		{"naive", &panel.NodeConfig{Protocol: "naive", ServerPort: 443}},
		{"socks", &panel.NodeConfig{Protocol: "socks", ServerPort: 1080}},
		{"http", &panel.NodeConfig{Protocol: "http", ServerPort: 8080}},
	}

	for _, tc := range protocols {
		t.Run(tc.name, func(t *testing.T) {
			cfg := buildConfig(config.KernelConfig{LogLevel: "warn"}, testNodeSpec(tc.nc), testUsers, kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
			data, err := json.Marshal(cfg)
			if err != nil {
				t.Fatalf("marshal %s: %v", tc.name, err)
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("round-trip %s: %v", tc.name, err)
			}
		})
	}
}

// --- Routes ---

func TestBuildRoutes_Default(t *testing.T) {
	route := buildRoutes(nil, nil, nil)
	assertMapValue(t, route, "final", "direct")

	rules := route["rules"].([]M)
	if len(rules) < 2 {
		t.Fatalf("expected at least 2 default rules, got %d", len(rules))
	}
	assertMapValue(t, rules[0], "action", "reject")
	assertMapValue(t, rules[1], "action", "reject")
}

func TestBuildRoutes_WithCustomRules(t *testing.T) {
	rules := []panel.RouteRule{
		{ID: 1, Match: []string{"blocked.com"}, Action: "block"},
		{ID: 2, Match: []string{"10.0.0.0/8"}, Action: "block"},
		{ID: 3, Match: []string{"allowed.com"}, Action: "direct"},
	}
	route := buildRoutes(testRouteRules(rules), nil, nil)
	allRules := route["rules"].([]M)

	if len(allRules) != 6 {
		t.Fatalf("rules count: got %d, want 6", len(allRules))
	}

	assertMapValue(t, allRules[0], "action", "sniff")
	if got := allRules[0]["network"].([]string); len(got) != 1 || got[0] != "tcp" {
		t.Fatalf("sniff rule must be TCP-only, got %v", got)
	}
	assertMapValue(t, allRules[3], "action", "reject")
	if _, ok := allRules[3]["domain_suffix"]; !ok {
		t.Error("domain rule should use domain_suffix")
	}

	assertMapValue(t, allRules[4], "action", "reject")
	if _, ok := allRules[4]["ip_cidr"]; !ok {
		t.Error("IP rule should use ip_cidr")
	}
}

func TestBuildRoutes_MultiMatch(t *testing.T) {
	// A single route with mixed domain + CIDR matches should produce separate rules
	// Wildcard *.evil.com should become evil.com for domain_suffix
	rules := []panel.RouteRule{
		{ID: 1, Match: []string{"*.evil.com", "bad.org", "192.168.1.0/24"}, Action: "block"},
		{ID: 2, Match: []string{"*.bypass.com"}, Action: "direct"},
	}
	route := buildRoutes(testRouteRules(rules), nil, nil)
	allRules := route["rules"].([]M)

	// sniff + 2 private-IP rules + 1 domain + 1 CIDR + 1 domain = 6
	if len(allRules) != 6 {
		t.Fatalf("rules count: got %d, want 6", len(allRules))
	}

	// Rule #2 (index 2): domains from first route (wildcards stripped)
	domains := allRules[3]["domain_suffix"].([]string)
	if len(domains) != 2 || domains[0] != "evil.com" || domains[1] != "bad.org" {
		t.Errorf("domain_suffix: got %v, want [evil.com bad.org]", domains)
	}
	assertMapValue(t, allRules[3], "action", "reject")

	// Rule #3 (index 3): CIDRs from first route
	cidrs := allRules[4]["ip_cidr"].([]string)
	if len(cidrs) != 1 || cidrs[0] != "192.168.1.0/24" {
		t.Errorf("ip_cidr: got %v, want [192.168.1.0/24]", cidrs)
	}
	assertMapValue(t, allRules[4], "action", "reject")

	// Rule #4 (index 4): direct rule (wildcard stripped)
	directDomains := allRules[5]["domain_suffix"].([]string)
	if len(directDomains) != 1 || directDomains[0] != "bypass.com" {
		t.Errorf("direct domain_suffix: got %v, want [bypass.com]", directDomains)
	}
	assertMapValue(t, allRules[5], "action", "route")
	assertMapValue(t, allRules[5], "outbound", "direct")
}

func TestBuildRoutes_WithCustomRouteRules(t *testing.T) {
	customRules := []model.CustomRouteRule{
		{
			Name: "direct-mixed",
			Match: model.RouteMatch{
				Domains:        []string{"full.example.com"},
				DomainSuffixes: []string{"example.org"},
				DomainKeywords: []string{"tracker"},
				DomainRegexes:  []string{`(^|\.)cdn[0-9]+\.example$`},
				IPCIDRs:        []string{"1.1.1.0/24"},
				Ports:          []string{"53", "1000-1002"},
				Networks:       []string{"tcp"},
				SourceCIDRs:    []string{"10.10.0.0/16"},
				SourcePorts:    []string{"2000-2001"},
				Protocols:      []string{"bittorrent"},
			},
			Action: model.RouteAction{Type: "direct"},
		},
	}
	route := buildRoutes(nil, customRules, nil)
	allRules := route["rules"].([]M)
	if len(allRules) != 10 {
		t.Fatalf("rules count: got %d, want 10", len(allRules))
	}
	assertMapValue(t, allRules[0], "action", "sniff")
	if allRules[1]["domain"].([]string)[0] != "full.example.com" || allRules[1]["domain_keyword"].([]string)[0] != "tracker" {
		t.Fatalf("unexpected domain rule: %v", allRules[1])
	}
	if allRules[1]["domain_suffix"].([]string)[0] != "example.org" || allRules[1]["domain_regex"].([]string)[0] == "" {
		t.Fatalf("unexpected extended domain rule: %v", allRules[1])
	}
	if allRules[2]["ip_cidr"].([]string)[0] != "1.1.1.0/24" {
		t.Fatalf("unexpected ip cidr rule: %v", allRules[2])
	}
	if allRules[3]["port"].([]int)[0] != 53 {
		t.Fatalf("unexpected port rule: %v", allRules[3])
	}
	if allRules[4]["network"].([]string)[0] != "tcp" {
		t.Fatalf("unexpected network rule: %v", allRules[4])
	}
	if allRules[5]["source_ip_cidr"].([]string)[0] != "10.10.0.0/16" {
		t.Fatalf("unexpected source cidr rule: %v", allRules[5])
	}
	if allRules[6]["source_port_range"].([]string)[0] != "2000:2001" {
		t.Fatalf("unexpected source port rule: %v", allRules[6])
	}
	if allRules[7]["protocol"].([]string)[0] != "bittorrent" {
		t.Fatalf("unexpected protocol rule: %v", allRules[7])
	}
}

func TestBuildRoutes_StructuredCustomRulesRemainFirst(t *testing.T) {
	raw := []map[string]any{{"domain": []string{"raw.example.com"}, "outbound": "raw-tag"}}
	custom := []model.CustomRouteRule{{
		Match:  model.RouteMatch{DomainSuffixes: []string{"structured.example"}},
		Action: model.RouteAction{Type: "direct"},
	}}
	route := buildRoutes(nil, custom, raw)
	allRules := route["rules"].([]M)
	if allRules[0]["action"] != "sniff" {
		t.Fatalf("expected sniff rule first, got %v", allRules[0])
	}
	if allRules[1]["outbound"] != "direct" {
		t.Fatalf("expected structured route after sniff, got %v", allRules[1])
	}
	if allRules[2]["outbound"] != "raw-tag" {
		t.Fatalf("expected raw custom route next, got %v", allRules[2]["outbound"])
	}
}

func TestBuildRoutes_StructuredPanelRules(t *testing.T) {
	allCriteria := model.RouteMatch{
		DomainSuffixes: []string{"api.example.com"},
		Ports:          []string{"443", "8000-8010"},
		Networks:       []string{"tcp"},
	}
	rules := []model.RouteRule{
		{
			ID:          2,
			Criteria:    &allCriteria,
			MatchMode:   "all",
			Priority:    20,
			Action:      "proxy",
			ActionValue: "warp",
		},
		{
			ID:        1,
			Criteria:  &model.RouteMatch{Domains: []string{"first.example.com"}},
			MatchMode: "any",
			Priority:  10,
			Action:    "direct",
		},
		{
			ID:       3,
			Criteria: &model.RouteMatch{Domains: []string{"disabled.example.com"}},
			Priority: 5,
			Disabled: true,
			Action:   "block",
		},
	}

	route := buildRoutes(rules, nil, nil)
	compiled := route["rules"].([]M)
	if len(compiled) != 5 {
		t.Fatalf("rules count: got %d, want 5", len(compiled))
	}
	if got := compiled[3]["domain"].([]string)[0]; got != "first.example.com" {
		t.Fatalf("priority ordering failed: %v", compiled[3])
	}
	last := compiled[4]
	if last["outbound"] != "warp" || last["type"] != "logical" || last["mode"] != "and" {
		t.Fatalf("unexpected all-mode rule: %#v", last)
	}
	logicalRules := last["rules"].([]M)
	if len(logicalRules) != 3 || logicalRules[2]["network"].([]string)[0] != "tcp" {
		t.Fatalf("all-mode logical matchers are incomplete: %#v", logicalRules)
	}
}

func TestBuildRoutes_GeoCollectionsAndNativeRuleSet(t *testing.T) {
	rules := []model.RouteRule{{
		Criteria: &model.RouteMatch{
			GeoIP:    []string{"cn", "private"},
			GeoSite:  []string{"category-ads-all"},
			RuleSets: []string{"cn_ip"},
		},
		Action: "block",
	}}
	route := buildRoutes(rules, nil, nil)
	compiled := route["rules"].([]M)
	if compiled[0]["action"] != "sniff" {
		t.Fatalf("geo site rules require sniffing: %#v", compiled[0])
	}
	if got := compiled[0]["network"].([]string); len(got) != 1 || got[0] != "tcp" {
		t.Fatalf("geo sniff rule must be TCP-only, got %v", got)
	}
	for _, item := range compiled[3:] {
		if item["action"] != "reject" {
			t.Fatalf("geo block must use reject action: %#v", item)
		}
	}
	if len(compiled) != 6 {
		t.Fatalf("compiled geo rule count: got %d, want 6: %#v", len(compiled), compiled)
	}
	if got := compiled[3]["rule_set"].([]string); len(got) != 1 || got[0] != "geosite-category-ads-all" {
		t.Fatalf("geosite matcher must reference its generated definition: %#v", compiled[3])
	}
	ipAlternatives := compiled[4]["rules"].([]M)
	if got := ipAlternatives[1]["rule_set"].([]string); len(got) != 1 || got[0] != "geoip-cn" {
		t.Fatalf("geoip matcher must reference its generated definition: %#v", compiled[4])
	}
	if got := compiled[5]["rule_set"].([]string); len(got) != 1 || got[0] != "cn_ip" {
		t.Fatalf("native rule-set matcher changed unexpectedly: %#v", compiled[5])
	}
	definitions := route["rule_set"].([]M)
	if len(definitions) != 2 {
		t.Fatalf("automatic definitions: got %d, want 2: %#v", len(definitions), definitions)
	}
	if definitions[0]["tag"] != "geoip-cn" || definitions[1]["tag"] != "geosite-category-ads-all" {
		t.Fatalf("unexpected automatic definitions: %#v", definitions)
	}
	if got := definitions[0]["url"].(string); got != "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/cn.srs" {
		t.Fatalf("MetaCubeX GeoIP URL = %q", got)
	}
	if got := definitions[1]["url"].(string); got != "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/category-ads-all.srs" {
		t.Fatalf("MetaCubeX GeoSite URL = %q", got)
	}
	data, err := json.Marshal(M{"route": route, "outbounds": []M{{"type": "direct", "tag": "direct"}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := include.Context(context.Background())
	if _, err := singJSON.UnmarshalExtendedContext[option.Options](ctx, data); err != nil {
		t.Fatalf("latest sing-box route options rejected generated rule-sets: %v\n%s", err, data)
	}
}

func TestMergeSingboxRuleSets_CustomDefinitionWins(t *testing.T) {
	generated := []M{
		{"type": "remote", "tag": "geoip-cn", "url": "https://generated/geoip-cn.srs"},
		{"type": "remote", "tag": "geosite-cn", "url": "https://generated/geosite-cn.srs"},
	}
	merged := mergeSingboxRuleSets(generated, []any{
		map[string]any{"type": "local", "tag": "geoip-cn", "path": "/etc/rules/cn.srs"},
	})
	if len(merged) != 2 || merged[0]["tag"] != "geosite-cn" || merged[1]["path"] != "/etc/rules/cn.srs" {
		t.Fatalf("custom rule-set did not replace generated tag: %#v", merged)
	}
}

func TestMergeSingboxExperimental_PreservesGeneratedCache(t *testing.T) {
	experimental := M{"cache_file": M{"enabled": true, "path": "/tmp/cache.db"}}
	mergeSingboxMap(experimental, map[string]any{"clash_api": map[string]any{"external_controller": "127.0.0.1:9090"}})
	cache := experimental["cache_file"].(map[string]any)
	if cache["enabled"] != true || cache["path"] != "/tmp/cache.db" {
		t.Fatalf("generated cache settings were lost: %#v", experimental)
	}
	if _, ok := experimental["clash_api"]; !ok {
		t.Fatalf("custom experimental settings were not merged: %#v", experimental)
	}
}

// --- TLS Config ---

func TestBuildTLSConfig_WithCert(t *testing.T) {
	nc := &panel.NodeConfig{ServerName: "example.com"}
	tls := buildTLSConfig(testNodeSpec(nc), kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, tls, "enabled", true)
	assertMapValue(t, tls, "server_name", "example.com")
	assertMapValue(t, tls, "certificate", []string{"CERT"})
	assertMapValue(t, tls, "key", []string{"KEY"})
}

func TestBuildTLSConfig_NoCert(t *testing.T) {
	nc := &panel.NodeConfig{ServerName: "example.com"}
	tls := buildTLSConfig(testNodeSpec(nc), kernel.TLSCert{})
	if tls != nil {
		t.Fatalf("expected nil tls config when cert paths are empty, got %+v", tls)
	}
}

func TestBuildTLSConfig_FallbackToHost(t *testing.T) {
	nc := &panel.NodeConfig{Host: "fallback.com"}
	tls := buildTLSConfig(testNodeSpec(nc), kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, tls, "server_name", "fallback.com")
}

func TestBuildTLSConfig_TLSSettingsOverride(t *testing.T) {
	nc := &panel.NodeConfig{
		ServerName: "original.com",
		TLSSettings: map[string]interface{}{
			"server_name": "override.com",
		},
	}
	tls := buildTLSConfig(testNodeSpec(nc), kernel.TLSCert{CertPEM: []byte("CERT"), KeyPEM: []byte("KEY")})
	assertMapValue(t, tls, "server_name", "override.com")
}

// --- Transport ---

func TestApplyTransport_TCP(t *testing.T) {
	base := M{}
	nc := &panel.NodeConfig{Network: "tcp"}
	applyTransport(base, testNodeSpec(nc))
	if _, exists := base["transport"]; exists {
		t.Error("tcp should not add transport")
	}
}

func TestApplyTransport_Empty(t *testing.T) {
	base := M{}
	nc := &panel.NodeConfig{Network: ""}
	applyTransport(base, testNodeSpec(nc))
	if _, exists := base["transport"]; exists {
		t.Error("empty network should not add transport")
	}
}

func TestApplyTransport_WS_MaxEarlyData(t *testing.T) {
	base := M{}
	nc := &panel.NodeConfig{
		Network: "ws",
		NetworkSettings: map[string]interface{}{
			"path":                   "/ws",
			"max_early_data":         2048,
			"early_data_header_name": "Sec-WebSocket-Protocol",
		},
	}
	applyTransport(base, testNodeSpec(nc))
	transport := base["transport"].(M)
	assertMapValue(t, transport, "max_early_data", 2048)
	assertMapValue(t, transport, "early_data_header_name", "Sec-WebSocket-Protocol")
}

// --- helpers ---

func assertMapValue(t *testing.T, m M, key string, expected interface{}) {
	t.Helper()
	val, ok := m[key]
	if !ok {
		t.Errorf("key %q not found in map", key)
		return
	}
	switch e := expected.(type) {
	case int:
		switch v := val.(type) {
		case int:
			if v != e {
				t.Errorf("%s: got %d, want %d", key, v, e)
			}
		case float64:
			if int(v) != e {
				t.Errorf("%s: got %v, want %d", key, v, e)
			}
		default:
			t.Errorf("%s: got %v (type %T), want %d", key, val, val, e)
		}
	default:
		if !reflect.DeepEqual(val, expected) {
			t.Errorf("%s: got %v, want %v", key, val, expected)
		}
	}
}

func TestExtractECHInbound(t *testing.T) {
	tests := []struct {
		name      string
		tls       map[string]interface{}
		expectNil bool
		expectKey []string
	}{
		{"no ech", map[string]interface{}{}, true, nil},
		{"disabled", map[string]interface{}{"ech": map[string]interface{}{"enabled": false}}, true, nil},
		{"enabled with key", map[string]interface{}{"ech": map[string]interface{}{
			"enabled": true,
			"key":     "-----BEGIN ECH KEYS-----\nABCD\n-----END ECH KEYS-----\n",
		}}, false, []string{"-----BEGIN ECH KEYS-----\nABCD\n-----END ECH KEYS-----\n"}},
		{"enabled with key_path", map[string]interface{}{"ech": map[string]interface{}{
			"enabled":  true,
			"key_path": "/etc/ech/keys.pem",
		}}, false, nil},
		{"enabled no key", map[string]interface{}{"ech": map[string]interface{}{"enabled": true}}, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractECHInbound(tt.tls)
			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result["enabled"] != true {
				t.Errorf("expected enabled=true, got %v", result["enabled"])
			}
			if tt.expectKey != nil {
				if !reflect.DeepEqual(result["key"], tt.expectKey) {
					t.Errorf("key: got %v, want %v", result["key"], tt.expectKey)
				}
			}
		})
	}
}
