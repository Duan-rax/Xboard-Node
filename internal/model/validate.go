package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cedar2025/xboard-node/internal/config"
)

func ValidateNodeSpec(n *NodeSpec, kcfg config.KernelConfig) error {
	if n == nil {
		return nil
	}

	effectiveKernelType := strings.TrimSpace(kcfg.Type)
	if effectiveKernelType == "" {
		effectiveKernelType = strings.TrimSpace(n.KernelType)
	}
	kernelType, err := normalizeKernelType(effectiveKernelType)
	if err != nil {
		return fmt.Errorf("normalize kernel type: %w", err)
	}

	additionalOutboundSources, err := collectAdditionalOutboundTagSources(kcfg.CustomConfig, kcfg.CustomOutbound)
	if err != nil {
		return fmt.Errorf("collect additional outbound tags: %w", err)
	}
	if err := validateOutboundTagCollisions(n.CustomOutbounds, additionalOutboundSources); err != nil {
		return fmt.Errorf("validate outbound tags: %w", err)
	}
	additionalTags := additionalTagNames(additionalOutboundSources)
	if err := validateEgressProxy(n.EgressProxy, kernelType, additionalTags); err != nil {
		return err
	}
	if n.EgressProxy != nil {
		for _, outbound := range n.CustomOutbounds {
			if strings.EqualFold(strings.TrimSpace(outbound.Tag), EgressProxyTag(n.EgressProxy.ID)) {
				return fmt.Errorf("validate egress proxy: generated tag %q conflicts with custom_outbounds", EgressProxyTag(n.EgressProxy.ID))
			}
		}
		additionalTags = append(additionalTags, EgressProxyTag(n.EgressProxy.ID))
	}
	availableTags := buildAvailableOutboundTags(n.CustomOutbounds, additionalTags)
	if err := ValidateCustomOutboundsForKernel(n.CustomOutbounds, kernelType, additionalTags); err != nil {
		return fmt.Errorf("validate custom outbounds: %w", err)
	}
	if err := ValidateCustomRouteRules(n.CustomRouteRules, kernelType, availableTags); err != nil {
		return fmt.Errorf("validate custom route rules: %w", err)
	}
	if err := ValidatePanelRouteRules(n.Routes, kernelType, availableTags); err != nil {
		return fmt.Errorf("validate panel route rules: %w", err)
	}
	if err := validateTransportKernel(n.Network, kernelType); err != nil {
		return err
	}
	return nil
}

func EgressProxyTag(id int) string {
	return fmt.Sprintf("panel-egress-%d", id)
}

func validateEgressProxy(proxy *EgressProxyConfig, kernelType string, existingTags []string) error {
	if proxy == nil {
		return nil
	}
	if proxy.ID <= 0 || strings.TrimSpace(proxy.Host) == "" || proxy.Port < 1 || proxy.Port > 65535 {
		return fmt.Errorf("validate egress proxy: id, host and port (1-65535) are required")
	}
	protocol := strings.ToLower(strings.TrimSpace(proxy.Protocol))
	if !map[string]bool{"socks5": true, "http": true, "shadowsocks": true, "vless": true, "trojan": true}[protocol] {
		return fmt.Errorf("validate egress proxy: protocol %q is not supported", proxy.Protocol)
	}
	if protocol == "shadowsocks" {
		if strings.TrimSpace(stringValue(proxy.Config, "method")) == "" {
			return fmt.Errorf("validate egress proxy: shadowsocks requires config.method")
		}
		if strings.TrimSpace(proxy.Password) == "" {
			return fmt.Errorf("validate egress proxy: shadowsocks requires a password")
		}
	}
	if protocol == "vless" {
		uuid := strings.TrimSpace(stringValue(proxy.Config, "uuid"))
		if uuid == "" {
			return fmt.Errorf("validate egress proxy: vless requires config.uuid")
		}
		if !uuidPattern.MatchString(uuid) {
			return fmt.Errorf("validate egress proxy: vless config.uuid must use the standard 36-character UUID format")
		}
	}
	if protocol == "trojan" && strings.TrimSpace(proxy.Password) == "" {
		return fmt.Errorf("validate egress proxy: trojan requires a password")
	}
	if reality, ok := mapValue(proxy.Config, "reality"); ok && len(reality) > 0 {
		if strings.TrimSpace(stringValue(reality, "public_key")) == "" {
			return fmt.Errorf("validate egress proxy: reality requires config.reality.public_key")
		}
		shortID := strings.TrimSpace(stringValue(reality, "short_id"))
		if !realityShortIDPattern.MatchString(shortID) {
			return fmt.Errorf("validate egress proxy: reality config.reality.short_id must be 2-16 even-length hexadecimal characters")
		}
		if strings.TrimSpace(proxy.TLSServerName) == "" && strings.TrimSpace(stringValue(proxy.Config, "server_name")) == "" {
			return fmt.Errorf("validate egress proxy: reality requires tls_server_name")
		}
	}
	tag := EgressProxyTag(proxy.ID)
	for _, existingTag := range existingTags {
		if strings.EqualFold(existingTag, tag) {
			return fmt.Errorf("validate egress proxy: generated tag %q conflicts with an existing outbound", tag)
		}
	}
	return nil
}

var (
	uuidPattern           = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	realityShortIDPattern = regexp.MustCompile(`(?i)^(?:[0-9a-f]{2}){1,8}$`)
)

func mapValue(values map[string]any, key string) (map[string]any, bool) {
	if values == nil {
		return nil, false
	}
	value, ok := values[key].(map[string]any)
	return value, ok
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

// singboxUnsupportedTransports lists transport types that sing-box does not support.
var singboxUnsupportedTransports = map[string]bool{
	"xhttp":     true,
	"splithttp": true,
}

func validateTransportKernel(network, kernelType string) error {
	net := strings.ToLower(strings.TrimSpace(network))
	if kernelType == "singbox" && singboxUnsupportedTransports[net] {
		return fmt.Errorf("transport %q is not supported by sing-box kernel; use xray kernel instead", net)
	}
	return nil
}

// ResolveKernelForTransport returns the kernel type required by the given
// transport. If the configured kernel cannot handle the transport, it returns
// the kernel that can. Otherwise it returns configuredKernel unchanged.
// This is used in machine mode to auto-switch kernel per node.
func ResolveKernelForTransport(network, configuredKernel string) string {
	net := strings.ToLower(strings.TrimSpace(network))
	if configuredKernel == "singbox" && singboxUnsupportedTransports[net] {
		return "xray"
	}
	return configuredKernel
}

func normalizeKernelType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "singbox", "sing-box":
		return "singbox", nil
	case "xray":
		return "xray", nil
	default:
		return "", fmt.Errorf("unsupported kernel type %q", value)
	}
}

func buildAvailableOutboundTags(structured []OutboundConfig, rawTags []string) map[string]struct{} {
	available := map[string]struct{}{
		"direct": {},
		"block":  {},
	}
	for _, outbound := range structured {
		tag := strings.ToLower(strings.TrimSpace(outbound.Tag))
		if tag != "" {
			available[tag] = struct{}{}
		}
	}
	for _, tag := range rawTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" {
			available[tag] = struct{}{}
		}
	}
	return available
}
