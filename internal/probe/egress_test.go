package probe

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		Protocol:      "vless",
		Host:          "203.0.113.10",
		Port:          443,
		TLSEnabled:    true,
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

func TestRunThroughRealSocksProxy(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ip":"203.0.113.9"}`))
	}))
	defer targetServer.Close()

	proxyServer := startTestSocksServer(t)
	proxyHost, proxyPortText, _ := net.SplitHostPort(proxyServer.Addr().String())
	proxyPort, _ := strconv.Atoi(proxyPortText)
	result := Run(context.Background(), model.EgressProbeRequest{
		RequestID: "probe-1",
		Target:    targetServer.URL,
		Proxy: &model.EgressProxyConfig{
			ID:       42,
			Protocol: "socks5",
			Host:     proxyHost,
			Port:     proxyPort,
		},
	})
	if !result.OK {
		t.Fatalf("probe failed: %s", result.Message)
	}
	if result.ExitIP != "203.0.113.9" {
		t.Fatalf("exit IP = %q", result.ExitIP)
	}
	if result.ProxyID != 42 || result.RequestID != "probe-1" {
		t.Fatalf("unexpected identity: %#v", result)
	}
}

func startTestSocksServer(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go serveTestSocksConnection(connection)
		}
	}()
	return listener
}

func serveTestSocksConnection(client net.Conn) {
	defer client.Close()
	header := make([]byte, 2)
	if _, err := io.ReadFull(client, header); err != nil || header[0] != 5 {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(client, methods); err != nil {
		return
	}
	_, _ = client.Write([]byte{5, 0})

	request := make([]byte, 4)
	if _, err := io.ReadFull(client, request); err != nil || request[0] != 5 || request[1] != 1 {
		return
	}
	var host string
	switch request[3] {
	case 1:
		address := make([]byte, 4)
		if _, err := io.ReadFull(client, address); err != nil {
			return
		}
		host = net.IP(address).String()
	case 3:
		length := make([]byte, 1)
		if _, err := io.ReadFull(client, length); err != nil {
			return
		}
		address := make([]byte, int(length[0]))
		if _, err := io.ReadFull(client, address); err != nil {
			return
		}
		host = string(address)
	case 4:
		address := make([]byte, 16)
		if _, err := io.ReadFull(client, address); err != nil {
			return
		}
		host = net.IP(address).String()
	default:
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(client, portBytes); err != nil {
		return
	}
	upstream, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(portBytes)))))
	if err != nil {
		_, _ = client.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer upstream.Close()
	_, _ = client.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
	go io.Copy(upstream, client)
	_, _ = io.Copy(client, upstream)
}
