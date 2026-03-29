package engine

import (
	"path"
	"strings"

	"github.com/ffreis/platform-guardian/internal/rule"
)

// Match filters rules based on scope matching logic for a given repo.
func Match(repo string, topics, languages []string, rules []*rule.Rule) []*rule.Rule {
	// Extract just the repo name (not org)
	repoName := repoShortName(repo)

	var matched []*rule.Rule
	for _, r := range rules {
		if matchesScope(r.Scope, repo, repoName, topics, languages) {
			matched = append(matched, r)
		}
	}
	return matched
}

func matchesScope(scope rule.ScopeSpec, repo, repoName string, topics, languages []string) bool {
	if isRepoExcluded(scope, repo, repoName) {
		return false
	}

	if !matchesAnyTopic(scope.Match.Topics, topics) {
		return false
	}

	if !matchesAnyLanguage(scope.Match.Languages, languages) {
		return false
	}

	return matchesNamePattern(scope.Match.NamePattern, repoName)
}

func repoShortName(repo string) string {
	if idx := strings.LastIndex(repo, "/"); idx >= 0 {
		return repo[idx+1:]
	}
	return repo
}

func isRepoExcluded(scope rule.ScopeSpec, repo, repoName string) bool {
	for _, excluded := range scope.Exclude.Repos {
		if excluded == repo {
			return true
		}
	}

	if scope.Exclude.NamePattern == "" {
		return false
	}

	matched, err := path.Match(scope.Exclude.NamePattern, repoName)
	return err == nil && matched
}

func matchesAnyTopic(required, provided []string) bool {
	// Empty required list = match all.
	if len(required) == 0 {
		return true
	}

	set := make(map[string]struct{}, len(provided))
	for _, t := range provided {
		set[t] = struct{}{}
	}
	for _, req := range required {
		if _, ok := set[req]; ok {
			return true
		}
	}
	return false
}

func matchesAnyLanguage(required, provided []string) bool {
	// Empty required list = match all.
	if len(required) == 0 {
		return true
	}

	normalized := make(map[string]struct{}, len(provided))
	for _, l := range provided {
		normalized[strings.ToLower(l)] = struct{}{}
	}
	for _, req := range required {
		if _, ok := normalized[strings.ToLower(req)]; ok {
			return true
		}
	}
	return false
}

func matchesNamePattern(pattern, repoName string) bool {
	// Empty pattern = match all.
	if pattern == "" {
		return true
	}

	matched, err := path.Match(pattern, repoName)
	return err == nil && matched
}
