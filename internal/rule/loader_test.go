package rule

import (
	"os"
	"path/filepath"
	"testing"
)

const testFilePerm = 0o644
const loadReturnedErrFmt = "Load returned error: %v"

func writeRuleFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), testFilePerm); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestLoadRule(t *testing.T) {
	dir := t.TempDir()
	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: test-rule-1
  name: Test Rule
  severity: error
  tags: [test]
spec:
  type: structure
  check:
    file_exists:
      path: README.md
  remediation:
    description: "Add a README"
`
	writeRuleFile(t, dir, "test.yaml", content)

	reg, err := Load([]string{dir})
	if err != nil {
		t.Fatalf(loadReturnedErrFmt, err)
	}

	rule, ok := reg.Rules["test-rule-1"]
	if !ok {
		t.Fatal("rule test-rule-1 not found in registry")
	}
	if rule.Name != "Test Rule" {
		t.Errorf("expected name 'Test Rule', got %q", rule.Name)
	}
	if rule.Severity != SeverityError {
		t.Errorf("expected severity error, got %q", rule.Severity)
	}
}

func TestLoadRuleSet(t *testing.T) {
	dir := t.TempDir()

	ruleContent := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: rule-a
  name: Rule A
  severity: warning
spec:
  type: structure
  check:
    file_exists:
      path: Makefile
`
	rsContent := `apiVersion: guardian/v1
kind: RuleSet
metadata:
  id: my-ruleset
  name: My RuleSet
spec:
  rules:
    - rule-a
`
	writeRuleFile(t, dir, "rule.yaml", ruleContent)
	writeRuleFile(t, dir, "ruleset.yaml", rsContent)

	reg, err := Load([]string{dir})
	if err != nil {
		t.Fatalf(loadReturnedErrFmt, err)
	}

	rs, ok := reg.RuleSets["my-ruleset"]
	if !ok {
		t.Fatal("ruleset my-ruleset not found in registry")
	}
	if len(rs.Rules) != 1 || rs.Rules[0] != "rule-a" {
		t.Errorf("unexpected ruleset rules: %v", rs.Rules)
	}
}

func TestLoadMultipleDocuments(t *testing.T) {
	dir := t.TempDir()
	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: doc-rule-1
  name: Doc Rule 1
  severity: error
spec:
  type: structure
  check:
    file_exists:
      path: file1.txt
---
apiVersion: guardian/v1
kind: Rule
metadata:
  id: doc-rule-2
  name: Doc Rule 2
  severity: warning
spec:
  type: structure
  check:
    file_exists:
      path: file2.txt
`
	writeRuleFile(t, dir, "multi.yaml", content)

	reg, err := Load([]string{dir})
	if err != nil {
		t.Fatalf(loadReturnedErrFmt, err)
	}

	if _, ok := reg.Rules["doc-rule-1"]; !ok {
		t.Error("doc-rule-1 not found")
	}
	if _, ok := reg.Rules["doc-rule-2"]; !ok {
		t.Error("doc-rule-2 not found")
	}
}

// TestLoadTFVariableRequired verifies the new tf_variable_required check type
// survives a full YAML → registry round-trip.
func TestLoadTFVariableRequired(t *testing.T) {
	dir := t.TempDir()
	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: tf-var-env-required
  name: Variable environment required
  severity: warning
spec:
  type: terraform
  check:
    tf_variable_required:
      name: environment
      type: string
  remediation:
    description: "Declare an environment variable"
`
	writeRuleFile(t, dir, "tfvar.yaml", content)
	reg, err := Load([]string{dir})
	if err != nil {
		t.Fatalf(loadReturnedErrFmt, err)
	}
	r, ok := reg.Rules["tf-var-env-required"]
	if !ok {
		t.Fatal("rule tf-var-env-required not found")
	}
	if r.Check.TFVariableReq == nil {
		t.Fatal("TFVariableReq check not parsed")
	}
	if r.Check.TFVariableReq.Name != "environment" {
		t.Errorf("expected name 'environment', got %q", r.Check.TFVariableReq.Name)
	}
	if r.Check.TFVariableReq.Type != "string" {
		t.Errorf("expected type 'string', got %q", r.Check.TFVariableReq.Type)
	}
}

// TestLoadFullRuleLibrary loads the entire rules/ directory and validates it
// to ensure all bundled rules are well-formed and cross-references resolve.
// This test uses a path relative to the module root and is skipped if the
// directory does not exist (e.g. in isolated unit test environments).
func TestLoadFullRuleLibrary(t *testing.T) {
	rulesDir := "../../rules"
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		t.Skipf("rules directory not found at %s", rulesDir)
	}

	reg, err := Load([]string{rulesDir})
	if err != nil {
		t.Fatalf(loadReturnedErrFmt, err)
	}

	if len(reg.Rules) == 0 {
		t.Fatal("no rules loaded from library")
	}
	if len(reg.RuleSets) == 0 {
		t.Fatal("no rulesets loaded from library")
	}
	if len(reg.Profiles) == 0 {
		t.Fatal("no profiles loaded from library")
	}

	errs := Validate(reg)
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("validation error: %v", e)
		}
		t.Fatalf("rule library has %d validation error(s)", len(errs))
	}

	t.Logf("loaded %d rules, %d rulesets, %d profiles", len(reg.Rules), len(reg.RuleSets), len(reg.Profiles))
}

// TestLoadCompositeRule verifies a composite AND check parses correctly.
func TestLoadCompositeRule(t *testing.T) {
	dir := t.TempDir()
	content := `apiVersion: guardian/v1
kind: Rule
metadata:
  id: composite-test
  name: Composite AND test
  severity: error
spec:
  type: structure
  check:
    composite:
      operator: AND
      checks:
        - file_exists:
            path: Makefile
        - file_exists:
            path: README.md
`
	writeRuleFile(t, dir, "composite.yaml", content)
	reg, err := Load([]string{dir})
	if err != nil {
		t.Fatalf(loadReturnedErrFmt, err)
	}
	r, ok := reg.Rules["composite-test"]
	if !ok {
		t.Fatal("composite-test rule not found")
	}
	if r.Check.Composite == nil {
		t.Fatal("composite check not parsed")
	}
	if r.Check.Composite.Operator != "AND" {
		t.Errorf("expected AND, got %q", r.Check.Composite.Operator)
	}
	if len(r.Check.Composite.Checks) != 2 {
		t.Errorf("expected 2 child checks, got %d", len(r.Check.Composite.Checks))
	}
}

// TestValidateInvalidCompositeOperator verifies that an invalid composite
// operator is caught by the validator, not silently accepted.
func TestValidateInvalidCompositeOperator(t *testing.T) {
	reg := NewRegistry()
	_ = reg.AddRule(&Rule{
		ID:       "bad-composite",
		Severity: SeverityError,
		Type:     RuleTypeStructure,
		Check: CheckSpec{
			Composite: &CompositeCheck{
				Operator: "XOR", // invalid
				Checks:   []CheckSpec{{FileExists: &FileExistsCheck{Path: "x"}}},
			},
		},
	})
	errs := Validate(reg)
	if len(errs) == 0 {
		t.Fatal("expected validation error for invalid composite operator")
	}
}

// TestEffectiveRules_ProfileMatch verifies that a profile with topic matching
// returns only its declared rules for a matching repo.
func TestEffectiveRulesProfileMatch(t *testing.T) {
	reg := NewRegistry()
	_ = reg.AddRule(&Rule{ID: "r1", Severity: SeverityError, Type: RuleTypeStructure,
		Check: CheckSpec{FileExists: &FileExistsCheck{Path: "x"}}})
	_ = reg.AddRule(&Rule{ID: "r2", Severity: SeverityWarning, Type: RuleTypeStructure,
		Check: CheckSpec{FileExists: &FileExistsCheck{Path: "y"}}})
	_ = reg.AddRuleSet(&RuleSet{ID: "rs1", Rules: []string{"r1"}})
	_ = reg.AddProfile(&Profile{
		ID:       "p1",
		Match:    ScopeMatch{Topics: []string{"terraform"}},
		RuleSets: []string{"rs1"},
	})

	// Repo with matching topic should get only r1.
	rules := reg.EffectiveRules("org/repo", []string{"terraform"}, nil)
	if len(rules) != 1 || rules[0].ID != "r1" {
		t.Errorf("expected [r1], got %v", ruleIDs(rules))
	}

	// Repo without matching topic falls back to all rules (r1, r2).
	rules = reg.EffectiveRules("org/other", nil, nil)
	if len(rules) != 2 {
		t.Errorf("expected 2 rules in fallback, got %d: %v", len(rules), ruleIDs(rules))
	}
}

// TestEffectiveRules_ProfileOverride verifies per-repo disable overrides.
func TestEffectiveRulesProfileOverride(t *testing.T) {
	reg := NewRegistry()
	_ = reg.AddRule(&Rule{ID: "r1", Severity: SeverityError, Type: RuleTypeStructure,
		Check: CheckSpec{FileExists: &FileExistsCheck{Path: "x"}}})
	_ = reg.AddRule(&Rule{ID: "r2", Severity: SeverityWarning, Type: RuleTypeStructure,
		Check: CheckSpec{FileExists: &FileExistsCheck{Path: "y"}}})
	_ = reg.AddRuleSet(&RuleSet{ID: "rs1", Rules: []string{"r1", "r2"}})
	_ = reg.AddProfile(&Profile{
		ID:       "p1",
		Match:    ScopeMatch{Topics: []string{}},
		RuleSets: []string{"rs1"},
		Overrides: []ProfileOverride{
			{Repo: "org/legacy", Disable: []string{"r2"}},
		},
	})

	// legacy repo should see only r1 (r2 disabled by override)
	rules := reg.EffectiveRules("org/legacy", nil, nil)
	for _, r := range rules {
		if r.ID == "r2" {
			t.Error("r2 should be disabled for org/legacy but was returned")
		}
	}

	// normal repo should see both
	rules = reg.EffectiveRules("org/normal", nil, nil)
	if len(rules) != 2 {
		t.Errorf("expected 2 rules for normal repo, got %d", len(rules))
	}
}

func ruleIDs(rules []*Rule) []string {
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}
	return ids
}

func TestValidateUnknownRuleInRuleSet(t *testing.T) {
	reg := NewRegistry()
	if err := reg.AddRuleSet(&RuleSet{
		ID:    "bad-ruleset",
		Name:  "Bad RuleSet",
		Rules: []string{"nonexistent-rule"},
	}); err != nil {
		t.Fatal(err)
	}

	errs := Validate(reg)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for unknown rule reference")
	}

	found := false
	for _, e := range errs {
		if e != nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("no non-nil errors returned")
	}
}
