package org

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

// panickyScanner panics on any repo whose name contains "boom" and returns
// an empty report otherwise. Used to inject panics into the worker pool.
type panickyScanner struct {
	calls int32
}

func (p *panickyScanner) Check(_ context.Context, opts engine.ScanOptions) (*engine.ScanReport, error) {
	atomic.AddInt32(&p.calls, 1)
	if contains(opts.Repo, "boom") {
		panic("simulated rule panic for " + opts.Repo)
	}
	return &engine.ScanReport{}, nil
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestWorkerPool_PanicDoesNotDeadlock is the panic-recovery contract test for
// the org-wide scanner. Before the fix, a panic inside a scanner.Check call
// would propagate through the goroutine, the Go runtime would crash the
// entire process (panic-in-goroutine semantics), and any concurrent scan
// would be aborted with no useful diagnostics.
//
// After the fix: panicking repos are surfaced as errors (logged), the worker
// continues processing the rest of the jobCh, and ScanAll returns within a
// reasonable bound.
func TestWorkerPoolPanicDoesNotDeadlock(t *testing.T) {
	p := &panickyScanner{}
	pool := NewWorkerPool(4, p, zap.NewNop())

	// Mix of safe and panicking repos. The panicking ones are scattered to
	// guarantee they hit different workers.
	repos := []RepoInfo{
		{FullName: "org/safe-1"},
		{FullName: "org/boom-1"},
		{FullName: "org/safe-2"},
		{FullName: "org/boom-2"},
		{FullName: "org/safe-3"},
		{FullName: "org/boom-3"},
		{FullName: "org/safe-4"},
	}

	done := make(chan struct{})
	var rep *engine.ScanReport
	var err error
	go func() {
		rep, err = pool.ScanAll(context.Background(), repos, "", rule.SeverityError)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ScanAll deadlocked: did not return within 5s after scanner panics")
	}

	if err != nil {
		t.Fatalf("ScanAll returned unexpected error: %v", err)
	}
	if rep == nil {
		t.Fatal("ScanAll returned nil report")
	}
	// Safe repos return empty Results slices; panicking repos go through the
	// error path. Either way every repo must have been attempted exactly once.
	if got := atomic.LoadInt32(&p.calls); got != int32(len(repos)) {
		t.Errorf("scanner.Check calls = %d, want %d (every repo must be attempted even after panics)", got, len(repos))
	}
}

// TestWorkerPool_AllPanicsSurfaceAsErrors verifies the edge case where every
// repo panics. The pool must drain without deadlocking and produce an empty
// report (every result was a panic).
func TestWorkerPoolAllPanicsSurfaceAsErrors(t *testing.T) {
	p := &panickyScanner{}
	pool := NewWorkerPool(2, p, zap.NewNop())

	repos := []RepoInfo{
		{FullName: "org/boom-1"},
		{FullName: "org/boom-2"},
		{FullName: "org/boom-3"},
	}

	done := make(chan struct{})
	var rep *engine.ScanReport
	var err error
	go func() {
		rep, err = pool.ScanAll(context.Background(), repos, "", rule.SeverityError)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ScanAll deadlocked when every repo panics")
	}

	if err != nil {
		t.Errorf("ScanAll returned err = %v, want nil (panics are per-repo failures, not pool failures)", err)
	}
	if rep == nil || len(rep.Results) != 0 {
		t.Errorf("rep.Results = %v, want empty (all panicked)", rep)
	}
}
