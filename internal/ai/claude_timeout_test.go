package ai

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

// TestEnhanceSuggestions_HangingServerHonorsContext verifies that a hung
// Anthropic endpoint cannot stall the scan indefinitely. The scan pipeline
// calls EnhanceSuggestions synchronously, so without a bounded deadline a
// dead-but-not-closed connection would freeze the entire org scan.
//
// We do not exercise the package-level 30s timeout here (too slow for a unit
// test). Instead we attach a short-deadline context and assert the request
// returns within that bound.
func TestEnhanceSuggestionsHangingServerHonorsContext(t *testing.T) {
	// Hung transport: blocks until the request's context is cancelled, then
	// returns the ctx error. This is exactly what a connection to a frozen
	// upstream looks like at the http.Client layer.
	origTransport := httpClient.Transport
	t.Cleanup(func() { httpClient.Transport = origTransport })
	httpClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	in := []Suggestion{{RuleID: "rule-1", Remediation: "do thing", AffectedRepos: []string{"o/r"}}}

	start := time.Now()
	out, err := EnhanceSuggestions(ctx, append([]Suggestion(nil), in...), "key-for-test")
	elapsed := time.Since(start)

	// EnhanceSuggestions wraps non-nil errors with "claude API:" and returns
	// the original suggestions unchanged. Either way the call must NOT exceed
	// the context deadline by a meaningful margin.
	if elapsed > 2*time.Second {
		t.Fatalf("EnhanceSuggestions did not honor 250ms context (elapsed=%s)", elapsed)
	}

	// The error path is the expected outcome: ctx cancelled mid-request.
	if err == nil {
		// Some HTTP clients return the original suggestions and a nil error if
		// the upstream is unreachable; but with our hung server we expect ctx
		// cancellation to surface as an error. Both behaviors imply the
		// suggestions array is untouched, which is the only invariant we
		// strictly require.
		if out[0].Enhanced != "" {
			t.Fatalf("expected Enhanced unset on hung-server failure, got %q", out[0].Enhanced)
		}
		return
	}
	// errors.Is is robust to wrapping; either DeadlineExceeded or Canceled is
	// acceptable.
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		// The transport may surface "context deadline exceeded" as a generic
		// url.Error wrapping a net.OpError. Best-effort string check as a
		// fallback so this test isn't brittle on transport internals.
		if !containsAny(err.Error(), "deadline", "canceled", "context") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// TestClaudeClient_HasTimeout pins the package contract: the default
// httpClient must always carry a non-zero Timeout. If anyone reinstates
// http.DefaultClient (which is timeout-less) this test fails immediately.
func TestClaudeClientHasTimeout(t *testing.T) {
	if httpClient.Timeout <= 0 {
		t.Fatalf("httpClient.Timeout = %v, want > 0 (must not be timeout-less)", httpClient.Timeout)
	}
	if httpClient.Timeout > 60*time.Second {
		t.Errorf("httpClient.Timeout = %v, suspiciously large (sane default ~30s)", httpClient.Timeout)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
