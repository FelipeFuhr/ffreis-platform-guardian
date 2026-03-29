package reporter

import "sort"

type ruleCount struct {
	id    string
	count int
}

func sortedRuleCounts(ruleCounts map[string]int) []ruleCount {
	rc := make([]ruleCount, 0, len(ruleCounts))
	for id, count := range ruleCounts {
		rc = append(rc, ruleCount{id: id, count: count})
	}
	sort.Slice(rc, func(i, j int) bool {
		if rc[i].count != rc[j].count {
			return rc[i].count > rc[j].count
		}
		return rc[i].id < rc[j].id
	})
	return rc
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
