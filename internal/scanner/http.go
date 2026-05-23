package scanner

import (
	"net/http"
	"time"
)

// HTTPClient is the bounded HTTP client used for all GitHub API calls in the
// scanner package. http.DefaultClient is intentionally not used: it has no
// timeout, so a slow or hung GitHub response would stall an org-wide scan
// indefinitely. The 30s ceiling is well above normal GitHub latency
// (typically sub-second) while preventing a misbehaving endpoint from
// blocking the scan pipeline.
//
// Tests in this package and downstream packages (engine) may swap this
// variable (or its Transport) to inject responses. Exported to allow that
// swapping from outside the scanner package — it is otherwise a knob users
// of the library may legitimately want to override (e.g. for a custom
// transport or instrumentation).
var HTTPClient = &http.Client{Timeout: 30 * time.Second}
