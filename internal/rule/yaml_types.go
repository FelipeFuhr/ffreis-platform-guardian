package rule

type ruleDocument struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   ruleMetadata `yaml:"metadata"`
	Spec       ruleSpec     `yaml:"spec"`
}
type ruleMetadata struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Severity string   `yaml:"severity"`
	Tags     []string `yaml:"tags"`
}
type ruleSpec struct {
	Type        string      `yaml:"type"`
	Scope       ScopeSpec   `yaml:"scope"`
	Check       CheckSpec   `yaml:"check"`
	Remediation Remediation `yaml:"remediation"`
	// For kind:RuleSet
	Rules []string `yaml:"rules"`
	// For kind:Profile
	Match     ScopeMatch        `yaml:"match"`
	RuleSets  []string          `yaml:"ruleSets"`
	Overrides []ProfileOverride `yaml:"overrides"`
}
