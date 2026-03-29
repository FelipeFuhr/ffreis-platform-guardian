package reporter

import "testing"

func TestSortedRuleCountsSortsByCountDescThenID(t *testing.T) {
	in := map[string]int{
		"b": 2,
		"a": 2,
		"c": 3,
	}
	out := sortedRuleCounts(in)
	if len(out) != 3 {
		t.Fatalf("expected 3, got %d", len(out))
	}
	if out[0].id != "c" || out[0].count != 3 {
		t.Fatalf("expected first=c(3), got %s(%d)", out[0].id, out[0].count)
	}
	if out[1].id != "a" || out[2].id != "b" {
		t.Fatalf("expected a then b for tied counts, got %s then %s", out[1].id, out[2].id)
	}
}
