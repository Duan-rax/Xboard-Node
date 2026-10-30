package xray

import (
	"sort"
	"strings"

	"github.com/cedar2025/xboard-node/internal/model"
)

func buildRouting(rules []model.RouteRule, customRouteRules []model.CustomRouteRule, customRules []map[string]any) M {
	var xrayRules []M
	rules = append([]model.RouteRule(nil), rules...)
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	// Structured custom routes now take the highest priority for panel-managed overrides.
	for _, rule := range customRouteRules {
		if rule.Disabled {
			continue
		}
		xrayRules = append(xrayRules, compileCustomRouteRule(rule)...)
	}

	// Raw custom routes remain the escape hatch, but no longer outrank structured rules.
	for _, cr := range customRules {
		xrayRules = append(xrayRules, M(cr))
	}

	xrayRules = append(xrayRules, M{
		"type": "field",
		"ip": []string{
			"10.0.0.0/8",
			"100.64.0.0/10",
			"127.0.0.0/8",
			"169.254.0.0/16",
			"172.16.0.0/12",
			"192.0.0.0/24",
			"192.168.0.0/16",
			"198.18.0.0/15",
			"fc00::/7",
			"fe80::/10",
			"::1/128",
		},
		"outboundTag": "block",
	})

	for _, rule := range rules {
		xrayRules = append(xrayRules, compilePanelRouteRule(rule)...)
	}

	return M{
		"domainStrategy": "AsIs",
		"rules":          xrayRules,
	}
}

func compilePanelRouteRule(rule model.RouteRule) []M {
	if rule.Disabled {
		return nil
	}
	match := panelRouteMatch(rule)
	action := panelRouteAction(rule)
	if strings.EqualFold(rule.MatchMode, "all") {
		return compilePanelRouteAll(match, action)
	}
	return compileCustomRouteRule(model.CustomRouteRule{Match: match, Action: action})
}

func panelRouteMatch(rule model.RouteRule) model.RouteMatch {
	if rule.Criteria != nil {
		return *rule.Criteria
	}
	match := model.RouteMatch{}
	for _, item := range rule.Match {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "geoip:") || strings.Contains(item, "/") {
			match.IPCIDRs = append(match.IPCIDRs, item)
			continue
		}
		if strings.HasPrefix(item, "geosite:") {
			match.Domains = append(match.Domains, item)
			continue
		}
		match.DomainSuffixes = append(match.DomainSuffixes, strings.TrimPrefix(item, "*."))
	}
	return match
}

func panelRouteAction(rule model.RouteRule) model.RouteAction {
	action := model.RouteAction{Type: "block"}
	switch rule.Action {
	case "direct":
		action.Type = "direct"
	case "proxy":
		action.Type = "route"
		action.Target = rule.ActionValue
	case "dns":
		action.Type = "route"
		action.Target = rule.ActionValue
		if action.Target == "" {
			action.Target = "dns-out"
		}
	}
	return action
}

func compilePanelRouteAll(match model.RouteMatch, action model.RouteAction) []M {
	entry := M{
		"type":        "field",
		"outboundTag": xrayOutboundForAction(action),
	}
	resultCount := 0
	if domains := xrayDomainMatchers(match); len(domains) > 0 {
		entry["domain"] = domains
		resultCount++
	}
	if len(match.IPCIDRs) > 0 || len(match.GeoIP) > 0 {
		ips := copyStrings(match.IPCIDRs)
		for _, value := range match.GeoIP {
			ips = append(ips, "geoip:"+strings.TrimPrefix(value, "geoip:"))
		}
		entry["ip"] = ips
		resultCount++
	}
	if len(match.Ports) > 0 {
		entry["port"] = strings.Join(match.Ports, ",")
		resultCount++
	}
	if len(match.Networks) > 0 {
		entry["network"] = strings.Join(match.Networks, ",")
		resultCount++
	}
	if len(match.SourceCIDRs) > 0 {
		entry["sourceIP"] = copyStrings(match.SourceCIDRs)
		resultCount++
	}
	if len(match.SourcePorts) > 0 {
		entry["sourcePort"] = strings.Join(match.SourcePorts, ",")
		resultCount++
	}
	if len(match.Protocols) > 0 {
		entry["protocol"] = copyStrings(match.Protocols)
		resultCount++
	}
	if resultCount == 0 {
		return nil
	}
	return []M{entry}
}

func compileCustomRouteRule(rule model.CustomRouteRule) []M {
	outbound := xrayOutboundForAction(rule.Action)
	var compiled []M

	if domains := xrayDomainMatchers(rule.Match); len(domains) > 0 {
		compiled = append(compiled, M{
			"type":        "field",
			"domain":      domains,
			"outboundTag": outbound,
		})
	}
	if len(rule.Match.IPCIDRs) > 0 || len(rule.Match.GeoIP) > 0 {
		ips := copyStrings(rule.Match.IPCIDRs)
		for _, value := range rule.Match.GeoIP {
			value = strings.TrimSpace(strings.TrimPrefix(value, "geoip:"))
			if value != "" {
				ips = append(ips, "geoip:"+value)
			}
		}
		compiled = append(compiled, M{
			"type":        "field",
			"ip":          ips,
			"outboundTag": outbound,
		})
	}
	if len(rule.Match.Ports) > 0 {
		compiled = append(compiled, M{
			"type":        "field",
			"port":        strings.Join(rule.Match.Ports, ","),
			"outboundTag": outbound,
		})
	}
	if len(rule.Match.Networks) > 0 {
		compiled = append(compiled, M{
			"type":        "field",
			"network":     strings.Join(rule.Match.Networks, ","),
			"outboundTag": outbound,
		})
	}
	if len(rule.Match.SourceCIDRs) > 0 {
		compiled = append(compiled, M{
			"type":        "field",
			"sourceIP":    copyStrings(rule.Match.SourceCIDRs),
			"outboundTag": outbound,
		})
	}
	if len(rule.Match.SourcePorts) > 0 {
		compiled = append(compiled, M{
			"type":        "field",
			"sourcePort":  strings.Join(rule.Match.SourcePorts, ","),
			"outboundTag": outbound,
		})
	}
	if len(rule.Match.Protocols) > 0 {
		compiled = append(compiled, M{
			"type":        "field",
			"protocol":    copyStrings(rule.Match.Protocols),
			"outboundTag": outbound,
		})
	}
	return compiled
}

func xrayDomainMatchers(match model.RouteMatch) []string {
	domains := make([]string, 0, len(match.Domains)+len(match.DomainSuffixes)+len(match.DomainKeywords)+len(match.DomainRegexes)+len(match.GeoSite))
	for _, value := range match.Domains {
		if value = strings.TrimSpace(strings.TrimPrefix(value, "full:")); value != "" {
			domains = append(domains, "full:"+value)
		}
	}
	for _, value := range match.DomainSuffixes {
		if value = strings.TrimSpace(strings.TrimPrefix(value, "domain:")); value != "" {
			domains = append(domains, "domain:"+strings.TrimPrefix(value, "*."))
		}
	}
	for _, value := range match.DomainKeywords {
		if value = strings.TrimSpace(strings.TrimPrefix(value, "keyword:")); value != "" {
			domains = append(domains, "keyword:"+value)
		}
	}
	for _, value := range match.DomainRegexes {
		if value = strings.TrimSpace(strings.TrimPrefix(value, "regexp:")); value != "" {
			domains = append(domains, "regexp:"+value)
		}
	}
	for _, value := range match.GeoSite {
		if value = strings.TrimSpace(strings.TrimPrefix(value, "geosite:")); value != "" {
			domains = append(domains, "geosite:"+value)
		}
	}
	return domains
}

func xrayOutboundForAction(action model.RouteAction) string {
	switch action.Type {
	case "direct":
		return "direct"
	case "route":
		if action.Target != "" {
			return action.Target
		}
		return "block"
	default:
		return "block"
	}
}

func copyStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}
