package org

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/engine"
	"github.com/ffreis/platform-guardian/internal/rule"
)

type WorkerPool struct {
	concurrency int
	engine      *engine.Engine
	log         *zap.Logger
}

func NewWorkerPool(concurrency int, eng *engine.Engine, log *zap.Logger) *WorkerPool {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &WorkerPool{
		concurrency: concurrency,
		engine:      eng,
		log:         log,
	}
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

				report, err := w.engine.Check(ctx, opts)
				if err != nil {
					w.log.Error("scan failed",
						zap.String("repo", repo.FullName),
						zap.Error(err),
					)
					resultCh <- nil
					continue
				}

				resultCh <- report.Results
			}
		}()
	}

	// Close result channel when all workers are done
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
