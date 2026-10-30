package model

func RouteSupportMatrix() map[string]KernelRouteSupport {
	return map[string]KernelRouteSupport{
		"xray": {
			Matchers: []string{
				"domains",
				"domain_suffixes",
				"domain_keywords",
				"domain_regexes",
				"ip_cidrs",
				"ports",
				"networks",
				"source_cidrs",
				"source_ports",
				"protocols",
				"geoip",
				"geosite",
			},
			Actions: []string{"block", "direct", "route"},
		},
		"singbox": {
			Matchers: []string{
				"domains",
				"domain_suffixes",
				"domain_keywords",
				"domain_regexes",
				"ip_cidrs",
				"ports",
				"networks",
				"source_cidrs",
				"source_ports",
				"protocols",
				"geoip",
				"geosite",
				"rule_sets",
			},
			Actions: []string{"block", "direct", "route"},
		},
	}
}

type KernelRouteSupport struct {
	Matchers []string
	Actions  []string
}
