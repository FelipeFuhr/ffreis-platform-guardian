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
		if severity == rule.SeverityWarning {
			level = "warning"
		} else if severity == rule.SeverityInfo {
			level = "notice"
		}

		fmt.Fprintf(r.w, "::%s title=%s::%s [%s]\n",
			level,
			ruleName,
			result.Message,
			result.Repo,
		)
	}
	return nil
}
