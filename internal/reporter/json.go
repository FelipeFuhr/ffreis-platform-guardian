package reporter

import (
	"encoding/json"
	"io"

	"github.com/ffreis/platform-guardian/internal/engine"
)

type JSONReporter struct {
	w io.Writer
}

func (r *JSONReporter) Report(report *engine.ScanReport) error {
	enc := json.NewEncoder(r.w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
