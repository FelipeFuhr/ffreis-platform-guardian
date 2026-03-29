package rule

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load walks dirs recursively for .yaml/.yml files, parses each document,
// and populates a registry. Returns all parse errors joined.
func Load(dirs []string) (*Registry, error) {
	registry := NewRegistry()
	var errs []error

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}

			fileErrs := loadFile(p, registry)
			errs = append(errs, fileErrs...)
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("walking %s: %w", dir, err))
		}
	}

	if len(errs) > 0 {
		return registry, errors.Join(errs...)
	}
	return registry, nil
}

func loadFile(path string, registry *Registry) []error {
	data, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("reading %s: %w", path, err)}
	}

	var errs []error
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var doc ruleDocument
		if err := decoder.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			errs = append(errs, fmt.Errorf("%s: decode error: %w", path, err))
			break
		}

		if doc.Kind == "" { // Skip empty documents.
			continue
		}

		if err := addDocument(registry, doc); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
		}
	}

	return errs
}

func addDocument(registry *Registry, doc ruleDocument) error {
	switch doc.Kind {
	case "Rule":
		return addRule(registry, doc)
	case "RuleSet":
		return addRuleSet(registry, doc)
	case "Profile":
		return addProfile(registry, doc)
	default:
		return fmt.Errorf("unknown kind: %s", doc.Kind)
	}
}

func addRule(registry *Registry, doc ruleDocument) error {
	r, err := parseRule(doc)
	if err != nil {
		return err
	}
	return registry.AddRule(r)
}

func addRuleSet(registry *Registry, doc ruleDocument) error {
	rs, err := parseRuleSet(doc)
	if err != nil {
		return err
	}
	return registry.AddRuleSet(rs)
}

func addProfile(registry *Registry, doc ruleDocument) error {
	p, err := parseProfile(doc)
	if err != nil {
		return err
	}
	return registry.AddProfile(p)
}

func parseRule(doc ruleDocument) (*Rule, error) {
	if doc.Metadata.ID == "" {
		return nil, fmt.Errorf("rule missing metadata.id")
	}

	return &Rule{
		ID:          doc.Metadata.ID,
		Name:        doc.Metadata.Name,
		Severity:    Severity(doc.Metadata.Severity),
		Tags:        doc.Metadata.Tags,
		Type:        RuleType(doc.Spec.Type),
		Scope:       doc.Spec.Scope,
		Check:       doc.Spec.Check,
		Remediation: doc.Spec.Remediation,
	}, nil
}

func parseRuleSet(doc ruleDocument) (*RuleSet, error) {
	if doc.Metadata.ID == "" {
		return nil, fmt.Errorf("ruleset missing metadata.id")
	}

	return &RuleSet{
		ID:    doc.Metadata.ID,
		Name:  doc.Metadata.Name,
		Rules: doc.Spec.Rules,
	}, nil
}

func parseProfile(doc ruleDocument) (*Profile, error) {
	if doc.Metadata.ID == "" {
		return nil, fmt.Errorf("profile missing metadata.id")
	}

	return &Profile{
		ID:        doc.Metadata.ID,
		Name:      doc.Metadata.Name,
		Match:     doc.Spec.Match,
		RuleSets:  doc.Spec.RuleSets,
		Overrides: doc.Spec.Overrides,
	}, nil
}
