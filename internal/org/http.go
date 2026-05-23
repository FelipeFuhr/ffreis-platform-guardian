package org

import (
	"net/http"
	"time"
)

// HTTPClient bounds outgoing GitHub API requests in the org package. See
// scanner/http.go for the rationale; the same constraint applies here:
// http.DefaultClient has no timeout, and an unbounded discovery request can
// stall the entire org scan loop.
//
// Tests may replace this variable (or its Transport) to inject canned
// responses.
var HTTPClient = &http.Client{Timeout: 30 * time.Second}
