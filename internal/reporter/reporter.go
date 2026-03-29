package reporter

import (
	"fmt"
	"io"

	"github.com/ffreis/platform-guardian/internal/engine"
)

type Reporter interface {
	Report(report *engine.ScanReport) error
}

func New(format string, w io.Writer) (Reporter, error) {
	switch format {
	case "table", "":
		return &TableReporter{w: w}, nil
	case "summary":
		return &SummaryReporter{w: w}, nil
	case "json":
		return &JSONReporter{w: w}, nil
	case "sarif":
		return &SARIFReporter{w: w}, nil
	case "github-annotations", "annotations":
		return &AnnotationsReporter{w: w}, nil
	default:
		return nil, fmt.Errorf("unknown format: %q (must be table, summary, json, sarif, or github-annotations)", format)
	}
}
