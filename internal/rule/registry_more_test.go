package rule

import "testing"

func TestRegistry_GetRule(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	r := &Rule{ID: "r1", Name: "R1"}
	if err := reg.AddRule(r); err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	got, ok := reg.GetRule("r1")
	if !ok || got == nil || got.ID != "r1" {
		t.Fatalf("expected to find rule r1")
	}
}

func TestMatchesAnyStringFold(t *testing.T) {
	t.Parallel()

	if !matchesAnyStringFold([]string{"Go"}, []string{"go", "terraform"}) {
		t.Fatalf("expected match (case-insensitive)")
	}
	if matchesAnyStringFold([]string{"python"}, []string{"go"}) {
		t.Fatalf("expected no match")
	}
}
