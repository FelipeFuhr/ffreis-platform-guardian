package ai

import (
	"strings"
	"testing"
)

const requireReadme = "require-readme"

func TestFormatAnalysis_PrintsPatternsAndSuggestions(t *testing.T) {
	t.Parallel()

	a := &Analysis{
		Patterns: []Pattern{
			{Kind: "systemic", Description: "Missing README.md across most repos"},
		},
		Suggestions: []Suggestion{
			{RuleID: requireReadme, Remediation: "Add README.md", AffectedRepos: []string{"org/a", "org/b"}},
		},
	}

	out := FormatAnalysis(a)
	if !strings.Contains(out, "Detected Patterns:") {
		t.Fatalf("expected patterns section, got:\n%s", out)
	}
	if !strings.Contains(out, "Fix Suggestions") {
		t.Fatalf("expected suggestions section, got:\n%s", out)
	}
	if !strings.Contains(out, requireReadme) {
		t.Fatalf("expected rule id to be included, got:\n%s", out)
	}
}
