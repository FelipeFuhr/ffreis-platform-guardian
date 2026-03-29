package engine

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/ffreis/platform-guardian/internal/rule"
	"github.com/ffreis/platform-guardian/internal/scanner"
)

type Engine struct {
	registry *rule.Registry
	log      *zap.Logger
}

func NewEngine(registry *rule.Registry, log *zap.Logger) *Engine {
	return &Engine{
		registry: registry,
		log:      log,
	}
}

type ScanOptions struct {
	Token    string
	Repo     string   // "org/repo"
	Topics   []string // passed in or fetched from API
	Language string
	Format   string
	FailOn   rule.Severity
}

func (e *Engine) Check(ctx context.Context, opts ScanOptions) (*ScanReport, error) {
	e.log.Info("starting check", zap.String("repo", opts.Repo))

	// Get effective rules for this repo
	languages := []string{}
	if opts.Language != "" {
		languages = []string{opts.Language}
	}
	effectiveRules := e.registry.EffectiveRules(opts.Repo, opts.Topics, languages)

	// Apply scope matching
	effectiveRules = Match(opts.Repo, opts.Topics, languages, effectiveRules)

	e.log.Info("rules to evaluate", zap.Int("count", len(effectiveRules)))

	// Build a snapshot
	snap := scanner.NewSnapshot(opts.Repo)

	// Run scanners as needed
	if err := e.populateSnapshot(ctx, opts, effectiveRules, snap); err != nil {
		return nil, fmt.Errorf("populating snapshot: %w", err)
	}

	// Evaluate each rule
	report := &ScanReport{
		RunID:       fmt.Sprintf("run-%d", time.Now().UnixNano()),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, r := range effectiveRules {
		result := Evaluate(r, snap)
		report.Results = append(report.Results, result)
		e.log.Debug("rule evaluated",
			zap.String("rule", r.ID),
			zap.String("status", string(result.Status)),
		)
	}

	return report, nil
}

func (e *Engine) populateSnapshot(ctx context.Context, opts ScanOptions, rules []*rule.Rule, snap *scanner.RepoSnapshot) error {
	needsStructure := false
	needsContent := false
	needsTerraform := false
	needsPolicy := false

	for _, r := range rules {
		switch r.Type {
		case rule.RuleTypeStructure:
			needsStructure = true
		case rule.RuleTypeContent:
			needsContent = true
			needsStructure = true
		case rule.RuleTypeTerraform:
			needsTerraform = true
		case rule.RuleTypePolicy:
			needsPolicy = true
		}
	}

	if opts.Token == "" {
		// Token is optional for public repos. Scanners handle empty tokens (unauthenticated API / clone).
		e.log.Info("no GitHub token provided, running scanners unauthenticated")
	}

	if needsStructure {
		s := scanner.NewStructureScanner(snap)
		if err := s.Scan(ctx, opts.Token, opts.Repo); err != nil {
			e.log.Warn("structure scanner failed", zap.Error(err))
		}
	}

	if needsContent {
		// Content scanner is lazy; individual files fetched on demand
		_ = needsContent
	}

	if needsTerraform {
		s := scanner.NewTerraformScanner(snap)
		if err := s.Scan(ctx, opts.Token, opts.Repo); err != nil {
			e.log.Warn("terraform scanner failed", zap.Error(err))
		}
	}

	if needsPolicy {
		if opts.Token == "" {
			e.log.Info("no GitHub token provided, skipping policy scanner")
			return nil
		}
		s := scanner.NewPolicyScanner(snap)
		if err := s.Scan(ctx, opts.Token, opts.Repo); err != nil {
			e.log.Warn("policy scanner failed", zap.Error(err))
		}
	}

	return nil
}
