package scanner

import (
	"context"
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

// githubGET builds and executes an authenticated GET request against the
// GitHub API. Callers are responsible for closing resp.Body and interpreting
// resp.StatusCode — that handling differs per endpoint (some treat 404 as a
// valid "not found" result, others don't) so it isn't folded in here.
//
// scan-fix(sonar:duplication): extracted from PolicyScanner's
// fetchRepoSettings/fetchBranchProtection/fetchTeamPermissions/
// fetchTeamRepoPermission, which each rebuilt an identical
// "new request, set auth header, set accept header" preamble — flagged by
// SonarCloud as new-code duplication once the errcheck cleanup made the
// deferred Close() lines identical too.
func githubGET(ctx context.Context, token, url, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set(httpHeaderAuthorization, authBearerPrefix+token)
	}
	req.Header.Set(httpHeaderAccept, accept)

	return HTTPClient.Do(req)
}
