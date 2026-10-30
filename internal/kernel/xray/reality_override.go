package xray

import (
	"fmt"
	"net"
	"strings"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/model"
)

// applyRealityRuntimeOverride keeps the panel destination untouched for
// subscription generation while allowing an Xray node to send Reality fallback
// traffic to a local PROXY-protocol-aware listener.
func applyRealityRuntimeOverride(inbound M, nc *model.NodeSpec, kcfg config.KernelConfig) {
	if kcfg.XrayRealityDestOverride == "" && kcfg.XrayRealityXver == 0 {
		return
	}
	stream, ok := inbound["streamSettings"].(M)
	if !ok {
		return
	}
	security, _ := stream["security"].(string)
	if security != "reality" {
		return
	}
	reality, ok := stream["realitySettings"].(M)
	if !ok {
		return
	}

	// Some panel versions expose only dest. Derive serverNames from the
	// original public destination before replacing the runtime destination.
	if _, hasServerNames := reality["serverNames"]; !hasServerNames {
		if serverName := realityPublicServerName(nc, reality["dest"]); serverName != "" {
			reality["serverNames"] = []string{serverName}
		}
	}
	if kcfg.XrayRealityDestOverride != "" {
		reality["dest"] = kcfg.XrayRealityDestOverride
	}
	if kcfg.XrayRealityXver != 0 {
		reality["xver"] = kcfg.XrayRealityXver
	}
}

func realityPublicServerName(nc *model.NodeSpec, generatedDest any) string {
	if nc.TLSSettings != nil {
		if serverName, ok := nc.TLSSettings["server_name"]; ok {
			if value := strings.TrimSpace(fmt.Sprintf("%v", serverName)); value != "" {
				return value
			}
		}
		if dest, ok := nc.TLSSettings["dest"]; ok {
			if value := realityHost(fmt.Sprintf("%v", dest)); value != "" {
				return value
			}
		}
	}
	return realityHost(fmt.Sprintf("%v", generatedDest))
}

func realityHost(destination string) string {
	destination = strings.TrimSpace(destination)
	if destination == "" || destination == "<nil>" {
		return ""
	}
	if host, _, err := net.SplitHostPort(destination); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(destination, "[]")
}
