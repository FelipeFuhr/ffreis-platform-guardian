package reporter

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
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

// writeTable writes a two-line header (label + separator) into a fresh
// tabwriter wrapping w, lets fill add the data rows, then flushes. Returns the
// first write or flush error encountered.
//
// scan-fix(sonar:S3776): extracted from summary.go/table.go — each report
// format built its own "tabwriter + header + rows + flush" block inline,
// which was both the source of the errcheck cleanup's duplication (SonarCloud
// flagged >3% duplication on new code) and pushed writeAggregateReport's
// cognitive complexity to 20 (over the 15 ceiling). One shared helper fixes
// both.
func writeTable(w io.Writer, header, separator string, fill func(tew *errWriter)) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	tew := &errWriter{w: tw}
	tew.println(header)
	tew.println(separator)
	fill(tew)
	if tew.err != nil {
		return tew.err
	}
	return tw.Flush()
}

// writeSeverityBreakdown writes header followed by one "  <severity>  <count>"
// line per non-empty severity bucket (error, warning, info, in that order)
// and a trailing blank line. No-op if sevBreakdown is empty.
//
// scan-fix(sonar:S3776/duplication): shared by summary.go's SummaryReporter
// and table.go's TableReporter — both built the identical block with only the
// header text differing.
func writeSeverityBreakdown(w io.Writer, header string, sevBreakdown map[string]int) error {
	if len(sevBreakdown) == 0 {
		return nil
	}

	ew := &errWriter{w: w}
	ew.println(header)
	for _, sev := range []string{"error", "warning", "info"} {
		if count, ok := sevBreakdown[sev]; ok {
			ew.printf("  %-10s %d\n", sev, count)
		}
	}
	ew.println()
	return ew.err
}

// writeTopRuleCounts writes header, then a two-column tabwriter table (via
// writeTable) of the top 10 entries in ruleCounts by count desc/id asc.
// No-op if ruleCounts is empty.
//
// scan-fix(sonar:duplication): shared by summary.go's writeMostViolatedRules
// and table.go's writeTopFailingRules — after the writeTable extraction above,
// both had converged on the identical body (sort+limit, header line, table,
// trailing blank line), differing only in the header/column text, which
// SonarCloud flagged as new-code duplication (22 lines in summary.go / 19 in
// table.go).
func writeTopRuleCounts(w io.Writer, header, tableHeader, tableSeparator string, ruleCounts map[string]int) error {
	if len(ruleCounts) == 0 {
		return nil
	}

	rc := sortedRuleCounts(ruleCounts)
	limit := minInt(10, len(rc))

	ew := &errWriter{w: w}
	ew.println(header)

	if err := writeTable(w, tableHeader, tableSeparator, func(tew *errWriter) {
		for _, entry := range rc[:limit] {
			tew.printf("  %s\t%d\n", entry.id, entry.count)
		}
	}); err != nil {
		return err
	}

	ew.println()
	return ew.err
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
