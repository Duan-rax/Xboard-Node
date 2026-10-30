package model

import (
	"fmt"
	"strings"
)

// ValidatePanelRouteRules protects the kernel from stale or manually crafted
// panel data. In particular, a globally visible outbound tag may not exist on
// every node to which a route group is bound.
func ValidatePanelRouteRules(rules []RouteRule, kernelType string, availableTags map[string]struct{}) error {
	for i, rule := range rules {
		if rule.Disabled {
			continue
		}
		match := RouteMatch{}
		if rule.Criteria != nil {
			match = *rule.Criteria
		} else if hasAnyNonBlank(rule.Match) {
			// Legacy matches are normalized by each kernel compiler. A sentinel
			// match lets the shared action validation run without reinterpreting
			// geoip:/geosite: values here.
			match.DomainSuffixes = []string{"legacy.panel.match"}
		}
		if !hasRouteMatch(match) {
			return fmt.Errorf("routes[%d].match is required", i)
		}

		actionName := strings.ToLower(strings.TrimSpace(rule.Action))
		switch actionName {
		case "dns":
			if err := validatePanelRouteMatch(i, match, kernelType); err != nil {
				return err
			}
		case "block", "direct":
			if err := validatePanelRouteMatch(i, match, kernelType); err != nil {
				return err
			}
		case "proxy":
			target := strings.TrimSpace(rule.ActionValue)
			if target == "" {
				return fmt.Errorf("routes[%d].action_value is required when action is proxy", i)
			}
			if _, ok := availableTags[strings.ToLower(target)]; !ok {
				return fmt.Errorf("routes[%d].action_value references unknown outbound %q", i, rule.ActionValue)
			}
			if err := validatePanelRouteMatch(i, match, kernelType); err != nil {
				return err
			}
		default:
			return fmt.Errorf("routes[%d].action %q is not supported", i, rule.Action)
		}
	}
	return nil
}

func validatePanelRouteMatch(index int, match RouteMatch, kernelType string) error {
	err := ValidateCustomRouteRules([]CustomRouteRule{{
		Match:  match,
		Action: RouteAction{Type: "direct"},
	}}, kernelType, nil)
	if err == nil {
		return nil
	}
	return fmt.Errorf("routes[%d]: %w", index, err)
}
