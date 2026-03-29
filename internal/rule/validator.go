package rule

import "fmt"

// Validate checks for consistency in the registry.
func Validate(r *Registry) []error {
	var errs []error

	errs = append(errs, validateRules(r)...)
	errs = append(errs, validateRuleSets(r)...)
	errs = append(errs, validateProfiles(r)...)

	return errs
}

func validateRules(r *Registry) []error {
	var errs []error

	for id, rule := range r.Rules {
		if id == "" || rule.ID == "" {
			errs = append(errs, fmt.Errorf("rule has empty ID"))
			continue
		}

		if !hasCheck(rule.Check) {
			errs = append(errs, fmt.Errorf("rule %s has no check defined", rule.ID))
		}

		if rule.Check.Composite == nil {
			continue
		}
		if err := validateComposite(rule.Check.Composite); err != nil {
			errs = append(errs, fmt.Errorf("rule %s: %w", rule.ID, err))
		}
	}

	return errs
}

func validateRuleSets(r *Registry) []error {
	var errs []error

	for _, rs := range r.RuleSets {
		for _, ruleID := range rs.Rules {
			if _, ok := r.Rules[ruleID]; ok {
				continue
			}
			errs = append(errs, fmt.Errorf("ruleset %s references unknown rule: %s", rs.ID, ruleID))
		}
	}
	return errs
}

func validateProfiles(r *Registry) []error {
	var errs []error

	for _, p := range r.Profiles {
		for _, rsID := range p.RuleSets {
			if _, ok := r.RuleSets[rsID]; ok {
				continue
			}
			errs = append(errs, fmt.Errorf("profile %s references unknown ruleset: %s", p.ID, rsID))
		}
	}
	return errs
}

func hasCheck(c CheckSpec) bool {
	return c.FileExists != nil ||
		c.FileAbsent != nil ||
		c.FileContains != nil ||
		c.FileNotContains != nil ||
		c.TFProviderReq != nil ||
		c.TFBackendConfig != nil ||
		c.TFRequiredTags != nil ||
		c.TFResourceForbid != nil ||
		c.TFModuleUsed != nil ||
		c.TFVariableReq != nil ||
		c.GHBranchProtect != nil ||
		c.GHTeamPermission != nil ||
		c.GHRepoSetting != nil ||
		c.Composite != nil
}

func validateComposite(c *CompositeCheck) error {
	switch c.Operator {
	case "AND", "OR", "NOT":
		// valid
	default:
		return fmt.Errorf("composite has invalid operator: %q (must be AND, OR, NOT)", c.Operator)
	}
	for _, child := range c.Checks {
		if child.Composite != nil {
			if err := validateComposite(child.Composite); err != nil {
				return err
			}
		}
	}
	return nil
}
