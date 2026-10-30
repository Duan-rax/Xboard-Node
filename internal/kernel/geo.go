package kernel

import (
	"strings"

	"github.com/cedar2025/xboard-node/internal/model"
)

// NeedsGeoIP returns true when any panel route rule contains a "geoip:" match entry.
func NeedsGeoIP(routes []model.RouteRule) bool {
	for _, r := range routes {
		if r.Criteria != nil && len(r.Criteria.GeoIP) > 0 {
			return true
		}
		for _, m := range r.Match {
			if strings.HasPrefix(m, "geoip:") {
				return true
			}
		}
	}
	return false
}

// NeedsGeoSite returns true when any panel route rule contains a "geosite:" match entry.
func NeedsGeoSite(routes []model.RouteRule) bool {
	for _, r := range routes {
		if r.Criteria != nil && len(r.Criteria.GeoSite) > 0 {
			return true
		}
		for _, m := range r.Match {
			if strings.HasPrefix(m, "geosite:") {
				return true
			}
		}
	}
	return false
}

// NeedsRuleSets returns true when structured panel routes reference native
// sing-box rule-set tags. Xray validation rejects these before compilation.
func NeedsRuleSets(routes []model.RouteRule) bool {
	for _, r := range routes {
		if r.Criteria != nil && len(r.Criteria.RuleSets) > 0 {
			return true
		}
	}
	return false
}

// NeedsSniff reports whether structured routes depend on a destination domain
// or a sniffed application protocol. IP/port-only rules do not need payload
// inspection and therefore avoid the sniffing delay.
func NeedsSniff(routes []model.RouteRule, customRules []model.CustomRouteRule) bool {
	matchNeedsSniff := func(match model.RouteMatch) bool {
		return len(match.Domains) > 0 || len(match.DomainSuffixes) > 0 ||
			len(match.DomainKeywords) > 0 || len(match.DomainRegexes) > 0 ||
			len(match.GeoSite) > 0 || len(match.Protocols) > 0
	}
	for _, route := range routes {
		if route.Disabled {
			continue
		}
		if route.Criteria != nil && matchNeedsSniff(*route.Criteria) {
			return true
		}
		for _, value := range route.Match {
			value = strings.TrimSpace(value)
			if value != "" && !strings.HasPrefix(value, "geoip:") && !strings.Contains(value, "/") {
				return true
			}
		}
	}
	for _, rule := range customRules {
		if !rule.Disabled && matchNeedsSniff(rule.Match) {
			return true
		}
	}
	return false
}
