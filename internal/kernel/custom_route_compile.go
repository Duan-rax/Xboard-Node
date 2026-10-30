package kernel

import (
	"strings"

	"github.com/cedar2025/xboard-node/internal/model"
)

func NeedsGeoIPRules(rules []model.CustomRouteRule) bool {
	for _, r := range rules {
		if len(r.Match.GeoIP) > 0 {
			return true
		}
		for _, v := range r.Match.IPCIDRs {
			if strings.HasPrefix(v, "geoip:") {
				return true
			}
		}
		for _, v := range r.Match.SourceCIDRs {
			if strings.HasPrefix(v, "geoip:") {
				return true
			}
		}
	}
	return false
}

func NeedsGeoSiteRules(rules []model.CustomRouteRule) bool {
	for _, r := range rules {
		if len(r.Match.GeoSite) > 0 {
			return true
		}
		for _, v := range r.Match.Domains {
			if strings.HasPrefix(v, "geosite:") {
				return true
			}
		}
		for _, v := range r.Match.DomainSuffixes {
			if strings.HasPrefix(v, "geosite:") {
				return true
			}
		}
	}
	return false
}

func NeedsRuleSetRules(rules []model.CustomRouteRule) bool {
	for _, r := range rules {
		if len(r.Match.RuleSets) > 0 {
			return true
		}
	}
	return false
}
