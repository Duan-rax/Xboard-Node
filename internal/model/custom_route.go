package model

type CustomRouteRule struct {
	Name     string
	Disabled bool
	Match    RouteMatch
	Action   RouteAction
}

type RouteMatch struct {
	Domains        []string
	DomainSuffixes []string
	DomainKeywords []string
	DomainRegexes  []string
	IPCIDRs        []string
	Ports          []string
	Networks       []string
	SourceCIDRs    []string
	SourcePorts    []string
	Protocols      []string
	GeoIP          []string
	GeoSite        []string
	RuleSets       []string
}

type RouteAction struct {
	Type   string
	Target string
}

func cloneCustomRouteRules(src []CustomRouteRule) []CustomRouteRule {
	if len(src) == 0 {
		return nil
	}
	out := make([]CustomRouteRule, 0, len(src))
	for _, rule := range src {
		out = append(out, CustomRouteRule{
			Name:     rule.Name,
			Disabled: rule.Disabled,
			Match:    cloneRouteMatch(rule.Match),
			Action:   cloneRouteAction(rule.Action),
		})
	}
	return out
}

func cloneRouteMatch(src RouteMatch) RouteMatch {
	return RouteMatch{
		Domains:        cloneStringSlice(src.Domains),
		DomainSuffixes: cloneStringSlice(src.DomainSuffixes),
		DomainKeywords: cloneStringSlice(src.DomainKeywords),
		DomainRegexes:  cloneStringSlice(src.DomainRegexes),
		IPCIDRs:        cloneStringSlice(src.IPCIDRs),
		Ports:          cloneStringSlice(src.Ports),
		Networks:       cloneStringSlice(src.Networks),
		SourceCIDRs:    cloneStringSlice(src.SourceCIDRs),
		SourcePorts:    cloneStringSlice(src.SourcePorts),
		Protocols:      cloneStringSlice(src.Protocols),
		GeoIP:          cloneStringSlice(src.GeoIP),
		GeoSite:        cloneStringSlice(src.GeoSite),
		RuleSets:       cloneStringSlice(src.RuleSets),
	}
}

func cloneRouteAction(src RouteAction) RouteAction {
	return RouteAction{
		Type:   src.Type,
		Target: src.Target,
	}
}
