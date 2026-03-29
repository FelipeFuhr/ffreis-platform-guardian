package ai

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestEnhanceSuggestions_NoAPIKeyIsNoOp(t *testing.T) {
	t.Parallel()

	in := []Suggestion{
		{RuleID: "rule-1", Remediation: "do the thing", AffectedRepos: []string{"org/repo"}},
	}

	out, err := EnhanceSuggestions(context.Background(), append([]Suggestion(nil), in...), "")
	if err != nil {
		t.Fatalf("EnhanceSuggestions() error = %v", err)
	}
	if out[0].Enhanced != "" {
		t.Fatalf("expected Enhanced to remain empty, got %q", out[0].Enhanced)
	}
}

func TestParseEnhanced_NumberedLines(t *testing.T) {
	t.Parallel()

	text := "1. First fix\n2. Second fix\n"
	got := parseEnhanced(text, 2)
	if got[0] != "First fix" || got[1] != "Second fix" {
		t.Fatalf("unexpected parse result: %#v", got)
	}
}

func TestEnhanceSuggestions_WithAPIKey_UsesClaudeResponse(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.String() != claudeAPIURL {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		if req.Header.Get("x-api-key") != "k" {
			t.Fatalf("expected x-api-key header")
		}
		body := `{"content":[{"type":"text","text":"1. Do X\\n"}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	in := []Suggestion{{RuleID: "rule-1", Remediation: "do the thing", AffectedRepos: []string{"org/repo"}}}
	out, err := EnhanceSuggestions(context.Background(), append([]Suggestion(nil), in...), "k")
	if err != nil {
		t.Fatalf("EnhanceSuggestions() error = %v", err)
	}
	if out[0].Enhanced == "" {
		t.Fatalf("expected Enhanced to be set")
	}
}

func TestBuildPrompt_IncludesRuleIDs(t *testing.T) {
	t.Parallel()

	p := buildPrompt([]Suggestion{
		{RuleID: "r1", Remediation: "Add README.md", AffectedRepos: []string{"org/a", "org/b"}},
	})
	if !strings.Contains(p, "Rule: r1") {
		t.Fatalf("expected prompt to include rule id, got:\n%s", p)
	}
}

func TestEnhanceSuggestions_EnvFallback(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"content":[{"type":"text","text":"1. Ok\\n"}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	old := os.Getenv("ANTHROPIC_API_KEY")
	_ = os.Setenv("ANTHROPIC_API_KEY", "envk")
	t.Cleanup(func() { _ = os.Setenv("ANTHROPIC_API_KEY", old) })

	in := []Suggestion{{RuleID: "rule-1", Remediation: "do the thing", AffectedRepos: []string{"org/repo"}}}
	out, err := EnhanceSuggestions(context.Background(), append([]Suggestion(nil), in...), "")
	if err != nil {
		t.Fatalf("EnhanceSuggestions() error = %v", err)
	}
	if out[0].Enhanced == "" {
		t.Fatalf("expected Enhanced to be set")
	}
}
