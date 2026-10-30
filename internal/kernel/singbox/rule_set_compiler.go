package singbox

import (
	"os"
	"sort"
	"strings"

	"github.com/cedar2025/xboard-node/internal/config"
	"github.com/cedar2025/xboard-node/internal/kernel"
	"github.com/cedar2025/xboard-node/internal/kernel/geodata"
	"github.com/cedar2025/xboard-node/internal/model"
)

func automaticGeoTags(prefix string, values []string) []string {
	tags := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.TrimPrefix(value, strings.TrimSuffix(prefix, "-")+":")
		if value != "" {
			tags = append(tags, prefix+value)
		}
	}
	return uniqueStrings(tags)
}

func logicalOr(rules []M) M {
	if len(rules) == 0 {
		return nil
	}
	if len(rules) == 1 {
		return rules[0]
	}
	return M{"type": "logical", "mode": "or", "rules": rules}
}

func automaticGeoRuleSets(panelRoutes []model.RouteRule, customRules []model.CustomRouteRule, source geodata.Source) []M {
	definitions := map[string]M{}
	add := func(match model.RouteMatch) {
		for _, value := range match.GeoIP {
			code := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "geoip:")))
			if code == "" || code == "private" {
				continue
			}
			tag := "geoip-" + code
			definitions[tag] = automaticGeoDefinition(source, "geoip", code)
		}
		for _, value := range match.GeoSite {
			code := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "geosite:")))
			if code == "" {
				continue
			}
			tag := "geosite-" + code
			definitions[tag] = automaticGeoDefinition(source, "geosite", code)
		}
	}
	for _, route := range panelRoutes {
		if !route.Disabled {
			add(panelRouteMatch(route))
		}
	}
	for _, rule := range customRules {
		if !rule.Disabled {
			add(rule.Match)
		}
	}
	tags := make([]string, 0, len(definitions))
	for tag := range definitions {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	result := make([]M, 0, len(tags))
	for _, tag := range tags {
		result = append(result, definitions[tag])
	}
	return result
}

func automaticGeoDefinition(source geodata.Source, kind, code string) M {
	tag := kind + "-" + code
	if path := geodata.RuleSetCachePath(source, kind, code); path != "" {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			return M{"type": "local", "tag": tag, "format": "binary", "path": path}
		}
	}
	urls := geodata.RuleSetURLs(source, kind, code)
	if len(urls) == 0 {
		return M{"type": "remote", "tag": tag, "format": "binary"}
	}
	return M{
		"type":            "remote",
		"tag":             tag,
		"format":          "binary",
		"url":             urls[0],
		"update_interval": "1d",
	}
}

func geoSource(kcfg config.KernelConfig) geodata.Source {
	cacheDir := strings.TrimSpace(kcfg.GeoDataDir)
	if cacheDir == "" {
		cacheDir = kcfg.ConfigDir
	}
	return geodata.NormalizeSource(geodata.Source{
		Provider:          kcfg.GeoRuleSetProvider,
		Variant:           kcfg.GeoDataVariant,
		GeoIPURL:          kcfg.GeoIPDataURL,
		GeoSiteURL:        kcfg.GeoSiteDataURL,
		GeoIPRuleSetURL:   kcfg.GeoIPRuleSetURL,
		GeoSiteRuleSetURL: kcfg.GeoSiteRuleSetURL,
		CacheDir:          cacheDir,
	})
}

func geoRuleRequirements(nc *model.NodeSpec) []geodata.Requirement {
	var requirements []geodata.Requirement
	add := func(match model.RouteMatch) {
		for _, value := range match.GeoIP {
			code := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "geoip:")))
			if code != "" && code != "private" {
				requirements = append(requirements, geodata.Requirement{Kind: "geoip", Tag: code})
			}
		}
		for _, value := range match.GeoSite {
			code := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "geosite:")))
			if code != "" {
				requirements = append(requirements, geodata.Requirement{Kind: "geosite", Tag: code})
			}
		}
	}
	for _, route := range nc.Routes {
		if !route.Disabled {
			add(panelRouteMatch(route))
		}
	}
	for _, rule := range nc.CustomRouteRules {
		if !rule.Disabled {
			add(rule.Match)
		}
	}
	return requirements
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mergeSingboxRuleSets(generated []M, custom any) []M {
	customItems := kernel.MergeAppendList(nil, custom)
	overridden := map[string]struct{}{}
	for _, item := range customItems {
		for _, tag := range singboxRuleSetTags(item["tag"]) {
			overridden[tag] = struct{}{}
		}
	}
	merged := make([]M, 0, len(generated)+len(customItems))
	for _, item := range generated {
		tags := singboxRuleSetTags(item["tag"])
		conflict := false
		for _, tag := range tags {
			if _, ok := overridden[tag]; ok {
				conflict = true
				break
			}
		}
		if !conflict {
			merged = append(merged, item)
		}
	}
	return append(merged, customItems...)
}

func singboxRuleSetTags(raw any) []string {
	switch value := raw.(type) {
	case string:
		if value != "" {
			return []string{value}
		}
	case []string:
		return value
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			if tag, ok := item.(string); ok && tag != "" {
				result = append(result, tag)
			}
		}
		return result
	}
	return nil
}
