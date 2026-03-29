package engine

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/rule"
)

func TestMatch_ExcludesRepo(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Exclude.Repos = []string{"acme/repo"}

	out := Match("acme/repo", nil, nil, []*rule.Rule{r})
	if len(out) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(out))
	}
}

func TestMatch_Topics_OR(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Match.Topics = []string{"terraform", "security"}

	out := Match("acme/repo", []string{"security"}, nil, []*rule.Rule{r})
	if len(out) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(out))
	}
}

func TestMatch_Languages_CaseInsensitive(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Match.Languages = []string{"Go"}

	out := Match("acme/repo", nil, []string{"go"}, []*rule.Rule{r})
	if len(out) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(out))
	}
}

func TestMatch_NamePattern(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Match.NamePattern = "repo-*"

	out := Match("acme/repo-1", nil, nil, []*rule.Rule{r})
	if len(out) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(out))
	}

	out = Match("acme/other", nil, nil, []*rule.Rule{r})
	if len(out) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(out))
	}
}
