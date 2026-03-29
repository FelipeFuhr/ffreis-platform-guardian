// Package ai provides pattern analysis and fix suggestions for Guardian scan reports.
// Pattern detection is purely algorithmic.  LLM-based suggestions are optional and
// require ANTHROPIC_API_KEY to be set (or an explicit key passed in).
package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

// Pattern describes an unusual or noteworthy drift pattern detected in the report.
type Pattern struct {
	// Kind classifies the pattern type.
	Kind string
	// Description is a human-readable summary.
	Description string
	// Repos affected by this pattern.
	Repos []string
	// Rules involved.
	Rules []string
	// Severity of the pattern.
	Severity string
}

// Suggestion is a fix suggestion for a group of related failures.
type Suggestion struct {
	// RuleID this suggestion addresses.
	RuleID string
	// AffectedRepos lists repos that fail this rule.
	AffectedRepos []string
	// Remediation is the rule's own remediation text.
	Remediation string
	// Enhanced is an LLM-generated enhancement (empty unless Claude is enabled).
	Enhanced string
}

// Analysis is the output of Analyze.
type Analysis struct {
	Patterns    []Pattern
	Suggestions []Suggestion
}

// Analyze detects drift patterns and builds prioritised fix suggestions from a
// ScanReport.  It is deterministic and requires no external services.
func Analyze(report *engine.ScanReport, registry ruleGetter) *Analysis {
	if len(report.Results) == 0 {
		return &Analysis{}
	}

	repoStats := report.RepoSummary()
	ruleCounts := report.RuleFailureCounts()
	totalRepos := report.RepoCount()

	analysis := &Analysis{}
	addSystemicViolations(analysis, report, ruleCounts, totalRepos)
	addWidespreadRepoFailures(analysis, repoStats)
	addCorrelatedFailures(analysis, report)
	addCompliantRepos(analysis, repoStats)

	analysis.Suggestions = buildSuggestions(report, ruleCounts)

	_ = registry
	return analysis
}

// ruleGetter is a minimal interface so Analyze can look up rule metadata.
type ruleGetter interface {
	GetRule(id string) (*rule.Rule, bool)
}

func addSystemicViolations(analysis *Analysis, report *engine.ScanReport, ruleCounts map[string]int, totalRepos int) {
	// A rule that fails in >50% of repos is systemic.
	if totalRepos <= 0 {
		return
	}

	for ruleID, count := range ruleCounts {
		pct := float64(count) / float64(totalRepos)
		if pct < 0.5 {
			continue
		}
		repos := reposFailingRule(report, ruleID)
		analysis.Patterns = append(analysis.Patterns, Pattern{
			Kind:        "systemic-violation",
			Description: fmt.Sprintf("Rule %q fails in %d/%d repos (%.0f%%) — likely a systemic gap rather than a per-repo issue", ruleID, count, totalRepos, pct*100),
			Repos:       repos,
			Rules:       []string{ruleID},
			Severity:    "high",
		})
	}
}

func addWidespreadRepoFailures(analysis *Analysis, repoStats map[string]*engine.RepoStats) {
	// Repos where >80% of checks fail.
	for repo, stats := range repoStats {
		if stats == nil {
			continue
		}
		total := stats.Pass + stats.Fail
		if total == 0 {
			continue
		}
		if float64(stats.Fail)/float64(total) < 0.8 {
			continue
		}
		analysis.Patterns = append(analysis.Patterns, Pattern{
			Kind:        "widespread-failure",
			Description: fmt.Sprintf("Repo %q fails %d/%d checks (%.0f%%) — may be newly onboarded or neglected", repo, stats.Fail, total, float64(stats.Fail)/float64(total)*100),
			Repos:       []string{repo},
			Severity:    "high",
		})
	}
}

func addCorrelatedFailures(analysis *Analysis, report *engine.ScanReport) {
	// Pairs of rules that always fail together across repos.
	for _, pair := range correlatedRules(report) {
		analysis.Patterns = append(analysis.Patterns, Pattern{
			Kind:        "correlated-failures",
			Description: fmt.Sprintf("Rules %q and %q always fail together — fixing one likely requires fixing both", pair[0], pair[1]),
			Rules:       pair,
			Severity:    "medium",
		})
	}
}

func addCompliantRepos(analysis *Analysis, repoStats map[string]*engine.RepoStats) {
	var compliantRepos []string
	for repo, s := range repoStats {
		if s != nil && s.Fail == 0 && s.Pass > 0 {
			compliantRepos = append(compliantRepos, repo)
		}
	}
	if len(compliantRepos) == 0 {
		return
	}

	sort.Strings(compliantRepos)
	analysis.Patterns = append(analysis.Patterns, Pattern{
		Kind:        "fully-compliant",
		Description: fmt.Sprintf("%d repo(s) pass all checks — use as reference implementations", len(compliantRepos)),
		Repos:       compliantRepos,
		Severity:    "info",
	})
}

type ruleFailCount struct {
	id    string
	count int
}

func buildSuggestions(report *engine.ScanReport, ruleCounts map[string]int) []Suggestion {
	ranked := rankRulesByFailures(ruleCounts)

	suggestions := make([]Suggestion, 0, len(ranked))
	for _, entry := range ranked {
		repos := reposFailingRule(report, entry.id)
		remediation := remediationForRule(report, entry.id)
		suggestions = append(suggestions, Suggestion{
			RuleID:        entry.id,
			AffectedRepos: repos,
			Remediation:   remediation,
		})
	}

	return suggestions
}

func rankRulesByFailures(ruleCounts map[string]int) []ruleFailCount {
	ranked := make([]ruleFailCount, 0, len(ruleCounts))
	for id, count := range ruleCounts {
		ranked = append(ranked, ruleFailCount{id, count})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].id < ranked[j].id
	})
	return ranked
}

func reposFailingRule(report *engine.ScanReport, ruleID string) []string {
	seen := map[string]bool{}
	for _, result := range report.Results {
		if result.Status == engine.StatusFail && result.Rule != nil && result.Rule.ID == ruleID {
			seen[result.Repo] = true
		}
	}
	repos := make([]string, 0, len(seen))
	for repo := range seen {
		repos = append(repos, repo)
	}
	sort.Strings(repos)
	return repos
}

func remediationForRule(report *engine.ScanReport, ruleID string) string {
	for _, result := range report.Results {
		if result.Rule != nil && result.Rule.ID == ruleID {
			return result.Remediation
		}
	}
	return ""
}

// correlatedRules returns pairs of rule IDs that always co-fail across repos.
func correlatedRules(report *engine.ScanReport) [][]string {
	repoFailRules := failingRulesByRepo(report)
	coCount, aCount := countRuleCoFailures(repoFailRules)
	correlated := correlatedPairs(coCount, aCount)
	sort.Slice(correlated, func(i, j int) bool {
		return strings.Join(correlated[i], ",") < strings.Join(correlated[j], ",")
	})
	return correlated
}

func failingRulesByRepo(report *engine.ScanReport) map[string]map[string]bool {
	repoFailRules := map[string]map[string]bool{}
	for _, result := range report.Results {
		if result.Status != engine.StatusFail || result.Rule == nil {
			continue
		}
		if repoFailRules[result.Repo] == nil {
			repoFailRules[result.Repo] = map[string]bool{}
		}
		repoFailRules[result.Repo][result.Rule.ID] = true
	}
	return repoFailRules
}

type rulePair struct{ a, b string }

func countRuleCoFailures(repoFailRules map[string]map[string]bool) (coCount map[rulePair]int, aCount map[string]int) {
	coCount = map[rulePair]int{}
	aCount = map[string]int{}

	for _, rules := range repoFailRules {
		ruleList := sortedRuleList(rules)
		for i, a := range ruleList {
			aCount[a]++
			for _, b := range ruleList[i+1:] {
				coCount[rulePair{a, b}]++
			}
		}
	}

	return coCount, aCount
}

func sortedRuleList(rules map[string]bool) []string {
	ruleList := make([]string, 0, len(rules))
	for r := range rules {
		ruleList = append(ruleList, r)
	}
	sort.Strings(ruleList)
	return ruleList
}

func correlatedPairs(coCount map[rulePair]int, aCount map[string]int) [][]string {
	// A pair is correlated if they co-fail in all their individual occurrences.
	correlated := make([][]string, 0, len(coCount))
	for p, count := range coCount {
		if count < 2 {
			continue
		}
		if count == aCount[p.a] && count == aCount[p.b] {
			correlated = append(correlated, []string{p.a, p.b})
		}
	}
	return correlated
}

// FormatAnalysis formats an Analysis as human-readable text for display.
func FormatAnalysis(a *Analysis) string {
	if len(a.Patterns) == 0 && len(a.Suggestions) == 0 {
		return "AI analysis: no patterns detected.\n"
	}

	var sb strings.Builder
	sb.WriteString("=== AI Analysis ===\n\n")

	if len(a.Patterns) > 0 {
		formatPatterns(&sb, a.Patterns)
	}

	if len(a.Suggestions) > 0 {
		formatSuggestions(&sb, a.Suggestions)
	}

	return sb.String()
}

func formatPatterns(sb *strings.Builder, patterns []Pattern) {
	sb.WriteString("Detected Patterns:\n")
	for i, p := range patterns {
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, p.Kind, p.Description))
	}
	sb.WriteString("\n")
}

func formatSuggestions(sb *strings.Builder, suggestions []Suggestion) {
	sb.WriteString("Fix Suggestions (ordered by impact):\n")
	for i, s := range suggestions {
		sb.WriteString(fmt.Sprintf("  %d. Rule: %s  (%d repos affected)\n", i+1, s.RuleID, len(s.AffectedRepos)))
		if s.Enhanced != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", s.Enhanced))
			continue
		}
		if s.Remediation != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", s.Remediation))
		}
		if len(s.AffectedRepos) > 0 && len(s.AffectedRepos) <= 5 {
			sb.WriteString(fmt.Sprintf("     Repos: %s\n", strings.Join(s.AffectedRepos, ", ")))
		}
	}
}
