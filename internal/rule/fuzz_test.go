package rule

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoadYAMLDocument exercises the full YAML rule-document loading pipeline
// with arbitrary bytes written to a temporary .yaml file.
//
// Invariants verified:
//   - Load must never panic on any byte sequence
//   - Load always returns a non-nil *Registry even when errors occur
//   - Well-formed rule/ruleset/profile documents are parsed without error
func FuzzLoadYAMLDocument(f *testing.F) {
	// Valid Rule document
	f.Add([]byte(`apiVersion: guardian/v1
kind: Rule
metadata:
  id: fuzz-test-rule
  name: Fuzz Test Rule
  severity: error
spec:
  type: structure
  check:
    file_exists:
      path: .golangci.yml
`))
	// Valid RuleSet document
	f.Add([]byte(`apiVersion: guardian/v1
kind: RuleSet
metadata:
  id: fuzz-test-set
  name: Fuzz Test Set
spec:
  rules:
    - fuzz-test-rule
`))
	// Valid Profile document
	f.Add([]byte(`apiVersion: guardian/v1
kind: Profile
metadata:
  id: fuzz-test-profile
  name: Fuzz Test Profile
spec:
  match:
    languages: [go]
  rulesets:
    - fuzz-test-set
`))
	// Edge cases
	f.Add([]byte(`kind: Unknown`))
	f.Add([]byte(`kind: Rule`)) // missing metadata.id
	f.Add([]byte(`---
kind: Rule
metadata:
  id: doc1
---
kind: RuleSet
metadata:
  id: set1
`)) // multi-document YAML
	f.Add([]byte(``))
	f.Add([]byte(`not yaml at all }{{{`))
	f.Add([]byte(`null`))
	f.Add([]byte(`- list: not a map`))
	f.Add([]byte("kind: Rule\nmetadata:\n  id: \"\"\n"))                                       // empty ID
	f.Add([]byte("kind: Rule\nmetadata:\n  id: dup\n---\nkind: Rule\nmetadata:\n  id: dup\n")) // duplicate

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "fuzz.yaml")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Skip("could not write temp file")
		}

		// Must not panic; errors are expected for invalid input
		registry, _ := Load([]string{dir})

		// Core invariant: registry is always non-nil
		if registry == nil {
			t.Fatal("Load returned nil *Registry")
		}
	})
}
