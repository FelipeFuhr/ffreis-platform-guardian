package rule

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

type Registry struct {
	Rules    map[string]*Rule
	RuleSets map[string]*RuleSet
	Profiles map[string]*Profile
}

func NewRegistry() *Registry {
	return &Registry{
		Rules:    make(map[string]*Rule),
		RuleSets: make(map[string]*RuleSet),
		Profiles: make(map[string]*Profile),
	}
}

// GetRule returns the Rule with the given ID, and whether it was found.
func (r *Registry) GetRule(id string) (*Rule, bool) {
	rule, ok := r.Rules[id]
	return rule, ok
}

func (r *Registry) AddRule(rule *Rule) error {
	if _, exists := r.Rules[rule.ID]; exists {
		return fmt.Errorf("duplicate rule ID: %s", rule.ID)
	}
	r.Rules[rule.ID] = rule
	return nil
}

func (r *Registry) AddRuleSet(rs *RuleSet) error {
	if _, exists := r.RuleSets[rs.ID]; exists {
		return fmt.Errorf("duplicate ruleset ID: %s", rs.ID)
	}
	r.RuleSets[rs.ID] = rs
	return nil
}

func (r *Registry) AddProfile(p *Profile) error {
	if _, exists := r.Profiles[p.ID]; exists {
		return fmt.Errorf("duplicate profile ID: %s", p.ID)
	}
	r.Profiles[p.ID] = p
	return nil
}

// EffectiveRules returns the rules that apply for a given repo.
func (r *Registry) EffectiveRules(repo string, repoTopics, repoLanguages []string) []*Rule {
	repoName := repoShortName(repo)

	matchedProfiles := r.matchingProfiles(repoName, repoTopics, repoLanguages)

	effectiveRuleIDs := r.effectiveRuleIDs(repo, matchedProfiles)
	disabledRules := disabledRuleIDs(repo, matchedProfiles)

	result := r.rulesByID(effectiveRuleIDs, disabledRules)

	// Sort by rule ID for determinism
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

func profileMatches(match ScopeMatch, repoName string, topics, languages []string) bool {
	if len(match.Topics) > 0 && !matchesAnyString(match.Topics, topics) {
		return false
	}
	if len(match.Languages) > 0 && !matchesAnyStringFold(match.Languages, languages) {
		return false
	}
	return matchesGlobOrAll(match.NamePattern, repoName)
}

func repoShortName(repo string) string {
	if idx := strings.LastIndex(repo, "/"); idx >= 0 {
		return repo[idx+1:]
	}
	return repo
}

func (r *Registry) matchingProfiles(repoName string, repoTopics, repoLanguages []string) []*Profile {
	var matchedProfiles []*Profile
	for _, p := range r.Profiles {
		if profileMatches(p.Match, repoName, repoTopics, repoLanguages) {
			matchedProfiles = append(matchedProfiles, p)
		}
	}
	return matchedProfiles
}

func (r *Registry) effectiveRuleIDs(repo string, matchedProfiles []*Profile) []string {
	seen := make(map[string]struct{})
	if len(matchedProfiles) == 0 {
		return r.allRuleIDs(seen)
	}

	return r.ruleIDsFromProfiles(seen, matchedProfiles)
}

func (r *Registry) allRuleIDs(seen map[string]struct{}) []string {
	var ids []string
	for id := range r.Rules {
		ids = appendIfNew(ids, seen, id)
	}
	return ids
}

func (r *Registry) ruleIDsFromProfiles(seen map[string]struct{}, matchedProfiles []*Profile) []string {
	var ids []string
	for _, p := range matchedProfiles {
		ids = r.appendProfileRuleIDs(ids, seen, p)
	}
	return ids
}

func (r *Registry) appendProfileRuleIDs(ids []string, seen map[string]struct{}, p *Profile) []string {
	for _, rsID := range p.RuleSets {
		rs, ok := r.RuleSets[rsID]
		if !ok {
			continue
		}
		for _, ruleID := range rs.Rules {
			ids = appendIfNew(ids, seen, ruleID)
		}
	}
	return ids
}

func appendIfNew(ids []string, seen map[string]struct{}, id string) []string {
	if _, ok := seen[id]; ok {
		return ids
	}
	seen[id] = struct{}{}
	return append(ids, id)
}

func disabledRuleIDs(repo string, matchedProfiles []*Profile) map[string]struct{} {
	disabled := make(map[string]struct{})
	for _, p := range matchedProfiles {
		for _, override := range p.Overrides {
			if override.Repo != repo {
				continue
			}
			for _, id := range override.Disable {
				disabled[id] = struct{}{}
			}
		}
	}
	return disabled
}

func (r *Registry) rulesByID(ruleIDs []string, disabled map[string]struct{}) []*Rule {
	var result []*Rule
	for _, id := range ruleIDs {
		if _, ok := disabled[id]; ok {
			continue
		}
		rule, ok := r.Rules[id]
		if !ok {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func matchesAnyString(required, provided []string) bool {
	set := make(map[string]struct{}, len(provided))
	for _, v := range provided {
		set[v] = struct{}{}
	}
	for _, req := range required {
		if _, ok := set[req]; ok {
			return true
		}
	}
	return false
}

func matchesAnyStringFold(required, provided []string) bool {
	set := make(map[string]struct{}, len(provided))
	for _, v := range provided {
		set[strings.ToLower(v)] = struct{}{}
	}
	for _, req := range required {
		if _, ok := set[strings.ToLower(req)]; ok {
			return true
		}
	}
	return false
}

func matchesGlobOrAll(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}
