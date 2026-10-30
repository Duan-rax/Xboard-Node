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
