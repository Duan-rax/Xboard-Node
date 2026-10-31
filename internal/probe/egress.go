package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	singJSON "github.com/sagernet/sing/common/json"

	"github.com/cedar2025/xboard-node/internal/model"
)

const (
	DefaultTarget = "https://api.ipify.org?format=json"
	probeTimeout  = 12 * time.Second
)

func Run(ctx context.Context, request model.EgressProbeRequest) model.EgressProbeResult {
	target := strings.TrimSpace(request.Target)
	if target == "" {
		target = DefaultTarget
	}
	result := model.EgressProbeResult{
		RequestID:   request.RequestID,
		Target:      target,
		CompletedAt: time.Now().Unix(),
	}
	if request.Proxy == nil {
		result.Message = "缺少出站代理配置"
		return result
	}
	result.ProxyID = request.Proxy.ID
	if err := validateTarget(target); err != nil {
		result.Message = err.Error()
		return result
	}
	if err := validateProxy(request.Proxy); err != nil {
		result.Message = err.Error()
		return result
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	probeCtx = include.Context(probeCtx)

	port, err := freePort()
	if err != nil {
		result.Message = "无法分配本地探测端口: " + err.Error()
		return result
	}
	data, err := json.Marshal(probeConfig(request.Proxy, port))
	if err != nil {
		result.Message = "生成出站探测配置失败: " + err.Error()
		return result
	}
	options, err := singJSON.UnmarshalExtendedContext[option.Options](probeCtx, data)
	if err != nil {
		result.Message = "解析出站探测配置失败: " + err.Error()
		return result
	}
	instance, err := box.New(box.Options{Context: probeCtx, Options: options})
	if err != nil {
		result.Message = "创建出站探测内核失败: " + err.Error()
		return result
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		result.Message = "启动出站探测内核失败: " + err.Error()
		return result
	}
	defer instance.Close()

	proxyURL := &url.URL{
		Scheme: "socks5",
		Host:   net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
	}
	result.TLSVerified = strings.HasPrefix(strings.ToLower(target), "https://")
	started := time.Now()
	client := &http.Client{Transport: transport, Timeout: probeTimeout}
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, target, nil)
	if err != nil {
		result.Message = "创建探测请求失败: " + err.Error()
		return result
	}
	req.Header.Set("User-Agent", "Xboard-Node-Egress-Probe/1.0")
	response, err := client.Do(req)
	result.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		result.Message = "代理请求失败: " + compactError(err)
		return result
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		result.Message = "读取代理响应失败: " + err.Error()
		return result
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		result.Message = fmt.Sprintf("代理响应异常（HTTP %d）", response.StatusCode)
		return result
	}

	result.OK = true
	result.ExitIP = parseExitIP(body)
	result.Message = fmt.Sprintf("节点侧代理链路可用（%d ms）", result.LatencyMS)
	if result.ExitIP != "" {
		result.Message += "，出口 IP " + result.ExitIP
	}
	if result.TLSVerified {
		result.Message += "，TLS 已建立"
	}
	return result
}

func validateTarget(raw string) error {
	if len(raw) > 2048 {
		return fmt.Errorf("探测目标过长")
	}
	target, err := url.Parse(raw)
	if err != nil || target.Host == "" {
		return fmt.Errorf("探测目标格式不正确")
	}
	if target.Scheme != "https" && target.Scheme != "http" {
		return fmt.Errorf("探测目标只支持 HTTP 或 HTTPS")
	}
	return nil
}

func validateProxy(proxy *model.EgressProxyConfig) error {
	if strings.TrimSpace(proxy.Host) == "" || proxy.Port < 1 || proxy.Port > 65535 {
		return fmt.Errorf("出站代理地址或端口无效")
	}
	switch strings.ToLower(proxy.Protocol) {
	case "socks5", "http", "shadowsocks", "vless", "trojan":
		return nil
	default:
		return fmt.Errorf("不支持的出站代理协议: %s", proxy.Protocol)
	}
}

func probeConfig(proxy *model.EgressProxyConfig, port int) map[string]any {
	protocol := strings.ToLower(proxy.Protocol)
	tag := "node-egress-probe-out"
	outbound := map[string]any{
		"type":        singboxType(protocol),
		"tag":         tag,
		"server":      proxy.Host,
		"server_port": proxy.Port,
	}
	switch protocol {
	case "socks5":
		outbound["version"] = "5"
		setUserPassword(outbound, proxy)
	case "http":
		setUserPassword(outbound, proxy)
	case "shadowsocks":
		outbound["method"] = stringValue(proxy.Config, "method")
		outbound["password"] = proxy.Password
	case "vless":
		outbound["uuid"] = stringValue(proxy.Config, "uuid")
		if flow := stringValue(proxy.Config, "flow"); flow != "" {
			outbound["flow"] = flow
		}
		if transport := transportConfig(proxy); transport != nil {
			outbound["transport"] = transport
		}
	case "trojan":
		outbound["password"] = proxy.Password
		if transport := transportConfig(proxy); transport != nil {
			outbound["transport"] = transport
		}
	}
	if tlsOptions := tlsConfig(proxy); tlsOptions != nil {
		outbound["tls"] = tlsOptions
	}
	return map[string]any{
		"log": map[string]any{"level": "error", "timestamp": false},
		"inbounds": []any{
			map[string]any{
				"type":        "mixed",
				"tag":         "node-egress-probe-in",
				"listen":      "127.0.0.1",
				"listen_port": port,
			},
		},
		"outbounds": []any{
			outbound,
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{"final": tag},
	}
}

func transportConfig(proxy *model.EgressProxyConfig) map[string]any {
	network := strings.ToLower(stringValue(proxy.Config, "network"))
	switch network {
	case "ws", "websocket":
		transport := map[string]any{"type": "ws", "path": firstNonEmpty(stringValue(proxy.Config, "path"), "/")}
		if host := stringValue(proxy.Config, "host"); host != "" {
			transport["headers"] = map[string]any{"Host": host}
		}
		return transport
	case "grpc":
		return map[string]any{"type": "grpc", "service_name": stringValue(proxy.Config, "service_name")}
	case "http":
		transport := map[string]any{"type": "http", "path": firstNonEmpty(stringValue(proxy.Config, "path"), "/")}
		if host := stringValue(proxy.Config, "host"); host != "" {
			transport["host"] = []string{host}
		}
		return transport
	case "httpupgrade":
		transport := map[string]any{"type": "httpupgrade", "path": firstNonEmpty(stringValue(proxy.Config, "path"), "/")}
		if host := stringValue(proxy.Config, "host"); host != "" {
			transport["host"] = host
		}
		return transport
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func setUserPassword(outbound map[string]any, proxy *model.EgressProxyConfig) {
	if proxy.Username != "" {
		outbound["username"] = proxy.Username
	}
	if proxy.Password != "" {
		outbound["password"] = proxy.Password
	}
}

func singboxType(protocol string) string {
	if protocol == "socks5" {
		return "socks"
	}
	return protocol
}

func tlsConfig(proxy *model.EgressProxyConfig) map[string]any {
	reality, _ := proxy.Config["reality"].(map[string]any)
	if !proxy.TLSEnabled && reality == nil {
		return nil
	}
	options := map[string]any{
		"enabled":  true,
		"insecure": proxy.AllowInsecure,
	}
	serverName := proxy.TLSServerName
	if serverName == "" {
		serverName = stringValue(proxy.Config, "server_name")
	}
	if serverName != "" {
		options["server_name"] = serverName
	}
	if reality != nil {
		options["reality"] = map[string]any{
			"enabled":    true,
			"public_key": reality["public_key"],
			"short_id":   reality["short_id"],
		}
		fingerprint := stringValue(reality, "fingerprint")
		if fingerprint == "" {
			fingerprint = stringValue(proxy.Config, "fingerprint")
		}
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		options["utls"] = map[string]any{"enabled": true, "fingerprint": fingerprint}
	}
	return options
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func parseExitIP(body []byte) string {
	var payload struct {
		IP string `json:"ip"`
	}
	if json.Unmarshal(body, &payload) == nil && net.ParseIP(payload.IP) != nil {
		return payload.IP
	}
	return ""
}

func compactError(err error) string {
	message := strings.ReplaceAll(strings.TrimSpace(err.Error()), "\n", " ")
	if len(message) > 300 {
		return message[:300] + "..."
	}
	return message
}
