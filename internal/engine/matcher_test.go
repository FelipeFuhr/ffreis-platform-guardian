package engine

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/rule"
)

const (
	matcherTestRepo    = "acme/repo"
	expectedOneRuleFmt = "expected 1 rule, got %d"
)

func TestMatchExcludesRepo(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Exclude.Repos = []string{matcherTestRepo}

	out := Match(matcherTestRepo, nil, nil, []*rule.Rule{r})
	if len(out) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(out))
	}
}

func TestMatchTopicsOR(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Match.Topics = []string{"terraform", "security"}

	out := Match(matcherTestRepo, []string{"security"}, nil, []*rule.Rule{r})
	if len(out) != 1 {
		t.Fatalf(expectedOneRuleFmt, len(out))
	}
}

func TestMatchLanguagesCaseInsensitive(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Match.Languages = []string{"Go"}

	out := Match(matcherTestRepo, nil, []string{"go"}, []*rule.Rule{r})
	if len(out) != 1 {
		t.Fatalf(expectedOneRuleFmt, len(out))
	}
}

func TestMatchNamePattern(t *testing.T) {
	r := &rule.Rule{ID: "r1"}
	r.Scope.Match.NamePattern = "repo-*"

	out := Match("acme/repo-1", nil, nil, []*rule.Rule{r})
	if len(out) != 1 {
		t.Fatalf(expectedOneRuleFmt, len(out))
	}

	out = Match("acme/other", nil, nil, []*rule.Rule{r})
	if len(out) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(out))
	}
}
