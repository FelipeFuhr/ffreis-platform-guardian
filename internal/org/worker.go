package org

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

// scanner is the minimal contract WorkerPool needs from the engine. Defined
// here so tests can substitute an implementation that returns errors or
// panics, exercising the panic-recovery contract below.
type scanner interface {
	Check(ctx context.Context, opts engine.ScanOptions) (*engine.ScanReport, error)
}

type WorkerPool struct {
	concurrency int
	scanner     scanner
	log         *zap.Logger
}

func NewWorkerPool(concurrency int, eng scanner, log *zap.Logger) *WorkerPool {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &WorkerPool{
		concurrency: concurrency,
		scanner:     eng,
		log:         log,
	}
}

// scanOne runs a single repo through the scanner with panic recovery. A
// misbehaving rule must NOT take down the worker goroutine — that would
// leak a permit from the worker pool (forever reducing throughput) and
// crash the whole process via the Go runtime's panic-in-goroutine default.
func (w *WorkerPool) scanOne(ctx context.Context, opts engine.ScanOptions) (results []engine.RuleResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("scanner panicked on %s: %v", opts.Repo, r)
		}
	}()

	report, err := w.scanner.Check(ctx, opts)
	if err != nil {
		return nil, err
	}
	return report.Results, nil
}

func (w *WorkerPool) ScanAll(ctx context.Context, repos []RepoInfo, token string, failOn rule.Severity) (*engine.ScanReport, error) {
	jobCh := make(chan RepoInfo, len(repos))
	resultCh := make(chan []engine.RuleResult, len(repos))

	// Feed all repos into the job channel
	for _, repo := range repos {
		jobCh <- repo
	}
	close(jobCh)

	var wg sync.WaitGroup
	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobCh {
				opts := engine.ScanOptions{
					Token:    token,
					Repo:     repo.FullName,
					Topics:   repo.Topics,
					Language: repo.Language,
					FailOn:   failOn,
				}

				results, err := w.scanOne(ctx, opts)
				if err != nil {
					w.log.Error("scan failed",
						zap.String("repo", repo.FullName),
						zap.Error(err),
					)
					resultCh <- nil
					continue
				}

				resultCh <- results
			}
		}()
	}

	// Close result channel when all workers are done. wg.Wait + close are
	// cheap; no defer-recover needed (channel close cannot panic here because
	// we own resultCh exclusively).
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Aggregate results
	combined := &engine.ScanReport{}
	for results := range resultCh {
		if results != nil {
			combined.Results = append(combined.Results, results...)
		}
	}

	return combined, nil
}
