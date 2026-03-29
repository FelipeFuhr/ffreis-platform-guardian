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
	Ref      string   // commit SHA or ref name
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

	// Policy checks require privileged GitHub API endpoints. If no token is
	// configured, skip policy rules entirely (structure/content/terraform checks
	// can still run unauthenticated on public repos).
	if opts.Token == "" {
		effectiveRules = filterPolicyRules(effectiveRules)
	}

	e.log.Info("rules to evaluate", zap.Int("count", len(effectiveRules)))

	// Build a snapshot
	snap := scanner.NewSnapshot(opts.Repo)
	snap.Ref = opts.Ref

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

func filterPolicyRules(rules []*rule.Rule) []*rule.Rule {
	filtered := rules[:0]
	for _, r := range rules {
		if r.Type == rule.RuleTypePolicy {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

type scanNeeds struct {
	structure bool
	content   bool
	terraform bool
	policy    bool
}

func computeScanNeeds(rules []*rule.Rule) scanNeeds {
	needs := scanNeeds{}
	for _, r := range rules {
		switch r.Type {
		case rule.RuleTypeStructure:
			needs.structure = true
		case rule.RuleTypeContent:
			needs.content = true
			needs.structure = true
		case rule.RuleTypeTerraform:
			needs.terraform = true
		case rule.RuleTypePolicy:
			needs.policy = true
		}
	}
	return needs
}

func (e *Engine) populateSnapshot(ctx context.Context, opts ScanOptions, rules []*rule.Rule, snap *scanner.RepoSnapshot) error {
	needs := computeScanNeeds(rules)

	if opts.Token == "" {
		// Token is optional for public repos. Scanners handle empty tokens (unauthenticated API / clone).
		e.log.Info("no GitHub token provided, running scanners unauthenticated")
	}

	if needs.structure {
		s := scanner.NewStructureScanner(snap)
		if err := s.Scan(ctx, opts.Token, opts.Repo); err != nil {
			e.log.Warn("structure scanner failed", zap.Error(err))
		}
	}

	if needs.content {
		// Collect the file paths referenced by all content rules and fetch them
		// eagerly so that FileContainsChecker/FileNotContainsChecker have data to
		// work with.  Files that 404 are stored as empty strings (treated as absent).
		for _, p := range contentPathsFromRules(rules) {
			content, err := scanner.FetchFile(ctx, opts.Token, opts.Repo, p)
			if err != nil {
				e.log.Warn("failed to fetch content file", zap.String("path", p), zap.Error(err))
				continue
			}
			snap.FileContents[p] = content
		}
	}

	if needs.terraform {
		s := scanner.NewTerraformScanner(snap)
		if err := s.Scan(ctx, opts.Token, opts.Repo); err != nil {
			e.log.Warn("terraform scanner failed", zap.Error(err))
		}
	}

	if needs.policy {
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

// contentPathsFromRules returns the deduplicated set of file paths referenced
// by all FileContains/FileNotContains checks in the given rules (including
// nested composite checks).
func contentPathsFromRules(rules []*rule.Rule) []string {
	seen := map[string]struct{}{}
	var paths []string
	for _, r := range rules {
		if r.Type != rule.RuleTypeContent {
			continue
		}
		for _, p := range contentPathsFromSpec(r.Check) {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// contentPathsFromSpec extracts file paths from a CheckSpec recursively.
func contentPathsFromSpec(spec rule.CheckSpec) []string {
	var paths []string
	if spec.FileContains != nil && spec.FileContains.Path != "" {
		paths = append(paths, spec.FileContains.Path)
	}
	if spec.FileNotContains != nil && spec.FileNotContains.Path != "" {
		paths = append(paths, spec.FileNotContains.Path)
	}
	if spec.Composite != nil {
		for _, child := range spec.Composite.Checks {
			paths = append(paths, contentPathsFromSpec(child)...)
		}
	}
	return paths
}
