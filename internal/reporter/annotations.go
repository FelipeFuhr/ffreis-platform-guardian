package reporter

import (
	"fmt"
	"io"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

type AnnotationsReporter struct {
	w io.Writer
}

func (r *AnnotationsReporter) Report(report *engine.ScanReport) error {
	for _, result := range report.Results {
		if result.Status != engine.StatusFail {
			continue
		}

		ruleName := ""
		severity := rule.SeverityError
		if result.Rule != nil {
			ruleName = result.Rule.Name
			severity = result.Rule.Severity
		}

		level := "error"
		// scan-fix(staticcheck:QF1003): tagged switch instead of if/else-if chain on severity
		switch severity {
		case rule.SeverityWarning:
			level = "warning"
		case rule.SeverityInfo:
			level = "notice"
		}

		if _, err := fmt.Fprintf(r.w, "::%s title=%s::%s [%s]\n",
			level,
			ruleName,
			result.Message,
			result.Repo,
		); err != nil {
			// scan-fix(golangci:errcheck): propagate the write failure instead of
			// silently dropping it — Report() already returns error for this.
			return err
		}
	}
	return nil
}
