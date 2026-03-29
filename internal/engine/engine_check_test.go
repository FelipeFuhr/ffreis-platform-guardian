package engine

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/rule"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestComputeScanNeeds(t *testing.T) {
	rules := []*rule.Rule{
		{Type: rule.RuleTypeStructure},
		{Type: rule.RuleTypeContent},
		{Type: rule.RuleTypeTerraform},
		{Type: rule.RuleTypePolicy},
	}

	needs := computeScanNeeds(rules)
	if !needs.structure || !needs.content || !needs.terraform || !needs.policy {
		t.Fatalf("expected all needs to be true, got %+v", needs)
	}
}

func TestEngineCheck_SkipsPolicyRulesWithoutToken(t *testing.T) {
	reg := rule.NewRegistry()
	_ = reg.AddRule(&rule.Rule{
		ID:       "policy-default-branch",
		Name:     "Default branch must be main",
		Severity: rule.SeverityError,
		Type:     rule.RuleTypePolicy,
		Check: rule.CheckSpec{
			GHRepoSetting: &rule.GHRepoSettingCheck{Field: "default_branch", Value: "main"},
		},
	})

	eng := NewEngine(reg, zap.NewNop())
	report, err := eng.Check(context.Background(), ScanOptions{Repo: "org/repo"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(report.Results) != 0 {
		t.Fatalf("expected 0 results (policy rules filtered), got %d", len(report.Results))
	}
}

func TestEngineCheck_StructureRuleRunsScannerAndEvaluates(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/git/trees/") {
			return httpResponse(http.StatusOK, `{"tree":[{"path":"README.md","type":"blob"}],"truncated":false}`), nil
		}
		return httpResponse(http.StatusNotFound, ""), nil
	})

	reg := rule.NewRegistry()
	_ = reg.AddRule(&rule.Rule{
		ID:       "require-readme",
		Name:     "README must exist",
		Severity: rule.SeverityError,
		Type:     rule.RuleTypeStructure,
		Check: rule.CheckSpec{
			FileExists: &rule.FileExistsCheck{Path: "README.md"},
		},
		Remediation: rule.Remediation{Description: "Add README.md"},
	})

	eng := NewEngine(reg, zap.NewNop())
	report, err := eng.Check(context.Background(), ScanOptions{Repo: "org/repo"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Status != StatusPass {
		t.Fatalf("expected Pass, got %s: %s", report.Results[0].Status, report.Results[0].Message)
	}
}
