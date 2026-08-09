package reporter

import (
	"fmt"
	"io"
	"sort"
)

// errWriter wraps an io.Writer and records the first write error it hits, so a
// sequence of Fprintf/Fprintln calls can be checked once at the end instead of
// after every call ("Errors are values" — https://go.dev/blog/errors-are-values).
//
// scan-fix(golangci:errcheck): introduced so summary.go/table.go can propagate a
// real write failure on the report writer (e.g. a full disk when writing a report
// to a file) instead of silently dropping it call-by-call.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) println(args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, args...)
}

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
