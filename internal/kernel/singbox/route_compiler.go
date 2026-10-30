package singbox

import (
	"sort"
	"strconv"
	"strings"

	"github.com/cedar2025/xboard-node/internal/kernel"
	"github.com/cedar2025/xboard-node/internal/kernel/geodata"
	"github.com/cedar2025/xboard-node/internal/model"
)

func buildRoutes(panelRoutes []model.RouteRule, customRules []model.CustomRouteRule, custom []map[string]any) M {
	return buildRoutesWithSource(panelRoutes, customRules, custom, geodata.Source{Provider: "metacubex", Variant: "full"})
}

func buildRoutesWithSource(panelRoutes []model.RouteRule, customRules []model.CustomRouteRule, custom []map[string]any, source geodata.Source) M {
	var rules []M
	panelRoutes = append([]model.RouteRule(nil), panelRoutes...)
	sort.SliceStable(panelRoutes, func(i, j int) bool {
		return panelRoutes[i].Priority < panelRoutes[j].Priority
	})
	if kernel.NeedsSniff(panelRoutes, customRules) {
		// The embedded sing-box build currently has an AnyTLS/UDP packet
		// nil-buffer crash after a route sniff action. Limit sniffing to TCP;
		// UDP routing still uses network, port, CIDR and source matchers.
		rules = append(rules, M{
			"network": []string{"tcp"},
			"action":  "sniff",
			"timeout": "300ms",
		})
	}

	// Structured custom routes now take the highest priority for panel-managed overrides.
	for _, rule := range customRules {
		if rule.Disabled {
			continue
		}
		rules = append(rules, compileCustomRouteRule(rule)...)
	}

	// Raw custom routes remain the escape hatch, but no longer outrank structured rules.
	for _, cr := range custom {
		rules = append(rules, M(cr))
	}

	// Standard blocks for private IPv4 and IPv6 ranges to prevent SSRF.
	rules = append(rules,
		M{
			"action": "reject",
			"ip_cidr": []string{
				"10.0.0.0/8",
				"100.64.0.0/10",
				"127.0.0.0/8",
				"169.254.0.0/16",
				"172.16.0.0/12",
				"192.0.0.0/24",
				"192.168.0.0/16",
				"198.18.0.0/15",
			},
		},
		M{
			"action": "reject",
			"ip_cidr": []string{
				"fc00::/7",
				"fe80::/10",
				"::1/128",
			},
		},
	)

	for _, pr := range panelRoutes {
		rules = append(rules, compilePanelRouteRule(pr)...)
	}

	route := M{
		"final": "direct",
		"rules": rules,
	}
	if ruleSets := automaticGeoRuleSets(panelRoutes, customRules, source); len(ruleSets) > 0 {
		route["rule_set"] = ruleSets
	}
	return route
}

func compilePanelRouteRule(pr model.RouteRule) []M {
	if pr.Disabled {
		return nil
	}

	match := panelRouteMatch(pr)
	action := panelRouteAction(pr)
	if strings.EqualFold(pr.MatchMode, "all") {
		return compilePanelRouteAll(match, action)
	}
	return compileCustomRouteRule(model.CustomRouteRule{Match: match, Action: action})
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
		action.Type = "dns"
	}
	return action
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
		if strings.HasPrefix(item, "geoip:") {
			match.GeoIP = append(match.GeoIP, strings.TrimPrefix(item, "geoip:"))
			continue
		}
		if strings.HasPrefix(item, "geosite:") {
			match.GeoSite = append(match.GeoSite, strings.TrimPrefix(item, "geosite:"))
			continue
		}
		item = strings.TrimPrefix(item, "*.")
		if strings.Contains(item, "/") {
			match.IPCIDRs = append(match.IPCIDRs, item)
		} else {
			match.DomainSuffixes = append(match.DomainSuffixes, item)
		}
	}
	return match
}

func compilePanelRouteAll(match model.RouteMatch, action model.RouteAction) []M {
	matchers := singboxMatchGroups(match)
	if len(matchers) == 0 {
		return nil
	}
	if len(matchers) == 1 {
		return []M{applySingboxAction(matchers[0], action)}
	}
	return []M{applySingboxAction(M{"type": "logical", "mode": "and", "rules": matchers}, action)}
}

func compileCustomRouteRule(rule model.CustomRouteRule) []M {
	if rule.Disabled {
		return nil
	}

	var compiled []M
	for _, matcher := range singboxMatchGroups(rule.Match) {
		compiled = append(compiled, applySingboxAction(matcher, rule.Action))
	}
	return compiled
}

func applySingboxAction(entry M, action model.RouteAction) M {
	switch action.Type {
	case "dns":
		entry["action"] = "hijack-dns"
	case "direct":
		entry["action"] = "route"
		entry["outbound"] = "direct"
	case "route":
		if action.Target == "" {
			entry["action"] = "reject"
		} else {
			entry["action"] = "route"
			entry["outbound"] = action.Target
		}
	default:
		entry["action"] = "reject"
	}
	return entry
}

func singboxMatchGroups(match model.RouteMatch) []M {
	groups := make([]M, 0, 8)
	if matcher := singboxDomainMatcher(match); matcher != nil {
		groups = append(groups, matcher)
	}
	if matcher := singboxIPMatcher(match); matcher != nil {
		groups = append(groups, matcher)
	}
	if len(match.Ports) > 0 {
		ports, ranges := splitPorts(match.Ports)
		matcher := M{}
		if len(ports) > 0 {
			matcher["port"] = ports
		}
		if len(ranges) > 0 {
			matcher["port_range"] = ranges
		}
		groups = append(groups, matcher)
	}
	if len(match.Networks) > 0 {
		groups = append(groups, M{"network": copyStrings(match.Networks)})
	}
	if len(match.SourceCIDRs) > 0 {
		groups = append(groups, M{"source_ip_cidr": copyStrings(match.SourceCIDRs)})
	}
	if len(match.SourcePorts) > 0 {
		ports, ranges := splitPorts(match.SourcePorts)
		matcher := M{}
		if len(ports) > 0 {
			matcher["source_port"] = ports
		}
		if len(ranges) > 0 {
			matcher["source_port_range"] = ranges
		}
		groups = append(groups, matcher)
	}
	if len(match.Protocols) > 0 {
		groups = append(groups, M{"protocol": copyStrings(match.Protocols)})
	}
	if len(match.RuleSets) > 0 {
		groups = append(groups, M{"rule_set": uniqueStrings(copyStrings(match.RuleSets))})
	}
	return groups
}

func singboxDomainMatcher(match model.RouteMatch) M {
	alternatives := make([]M, 0, 2)
	direct := M{}
	if len(match.Domains) > 0 {
		direct["domain"] = copyStrings(match.Domains)
	}
	if len(match.DomainSuffixes) > 0 {
		direct["domain_suffix"] = copyStrings(match.DomainSuffixes)
	}
	if len(match.DomainKeywords) > 0 {
		direct["domain_keyword"] = copyStrings(match.DomainKeywords)
	}
	if len(match.DomainRegexes) > 0 {
		direct["domain_regex"] = copyStrings(match.DomainRegexes)
	}
	if len(direct) > 0 {
		alternatives = append(alternatives, direct)
	}
	if tags := automaticGeoTags("geosite-", match.GeoSite); len(tags) > 0 {
		alternatives = append(alternatives, M{"rule_set": tags})
	}
	return logicalOr(alternatives)
}

func singboxIPMatcher(match model.RouteMatch) M {
	alternatives := make([]M, 0, 2)
	direct := M{}
	if len(match.IPCIDRs) > 0 {
		direct["ip_cidr"] = copyStrings(match.IPCIDRs)
	}
	geoValues := make([]string, 0, len(match.GeoIP))
	for _, value := range match.GeoIP {
		value = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "geoip:")))
		if value == "private" {
			direct["ip_is_private"] = true
		} else if value != "" {
			geoValues = append(geoValues, value)
		}
	}
	if len(direct) > 0 {
		alternatives = append(alternatives, direct)
	}
	if tags := automaticGeoTags("geoip-", geoValues); len(tags) > 0 {
		alternatives = append(alternatives, M{"rule_set": tags})
	}
	return logicalOr(alternatives)
}

func splitPorts(values []string) ([]int, []string) {
	var ports []int
	var ranges []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.ContainsAny(value, ":-") {
			ranges = append(ranges, strings.ReplaceAll(value, "-", ":"))
			continue
		}
		if port, err := strconv.Atoi(value); err == nil {
			ports = append(ports, port)
		}
	}
	return ports, ranges
}

func copyStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}
