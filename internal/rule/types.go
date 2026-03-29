package rule

// Severity of a rule finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// RuleType determines which scanner and check implementation to use.
type RuleType string

const (
	RuleTypeStructure RuleType = "structure"
	RuleTypeContent   RuleType = "content"
	RuleTypeTerraform RuleType = "terraform"
	RuleTypePolicy    RuleType = "policy"
)

// CheckSpec is the raw YAML check block — only one field will be non-nil at a time.
type CheckSpec struct {
	FileExists       *FileExistsCheck       `yaml:"file_exists,omitempty"`
	FileAbsent       *FileAbsentCheck       `yaml:"file_absent,omitempty"`
	FileContains     *FileContainsCheck     `yaml:"file_contains,omitempty"`
	FileNotContains  *FileNotContainsCheck  `yaml:"file_not_contains,omitempty"`
	TFProviderReq    *TFProviderReqCheck    `yaml:"tf_provider_required,omitempty"`
	TFBackendConfig  *TFBackendConfigCheck  `yaml:"tf_backend_config,omitempty"`
	TFRequiredTags   *TFRequiredTagsCheck   `yaml:"tf_required_tags,omitempty"`
	TFResourceForbid *TFResourceForbidCheck `yaml:"tf_resource_forbidden,omitempty"`
	TFModuleUsed     *TFModuleUsedCheck     `yaml:"tf_module_used,omitempty"`
	TFVariableReq    *TFVariableReqCheck    `yaml:"tf_variable_required,omitempty"`
	GHBranchProtect  *GHBranchProtectCheck  `yaml:"gh_branch_protection,omitempty"`
	GHTeamPermission *GHTeamPermissionCheck `yaml:"gh_team_permission,omitempty"`
	GHRepoSetting    *GHRepoSettingCheck    `yaml:"gh_repo_setting,omitempty"`
	Composite        *CompositeCheck        `yaml:"composite,omitempty"`
}

type FileExistsCheck struct {
	Path string `yaml:"path"`
}
type FileAbsentCheck struct {
	Path string `yaml:"path"`
}
type FileContainsCheck struct {
	Path    string `yaml:"path"`
	Pattern string `yaml:"pattern"`
}
type FileNotContainsCheck struct {
	Path    string `yaml:"path"`
	Pattern string `yaml:"pattern"`
}
type TFProviderReqCheck struct {
	Source  string `yaml:"source"`
	Version string `yaml:"version"`
}
type TFBackendConfigCheck struct {
	Type   string            `yaml:"type"`
	Fields map[string]string `yaml:"fields"`
}
type TFRequiredTagsCheck struct {
	Tags []string `yaml:"tags"`
}
type TFResourceForbidCheck struct {
	Type string `yaml:"type"`
}
type TFModuleUsedCheck struct {
	Source string `yaml:"source"`
}
type TFVariableReqCheck struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // optional; if set, variable must declare this type
}
type GHBranchProtectCheck struct {
	Branch              string `yaml:"branch"`
	RequirePRReviews    bool   `yaml:"require_pr_reviews"`
	RequireStatusChecks bool   `yaml:"require_status_checks"`
}
type GHTeamPermissionCheck struct {
	Team       string `yaml:"team"`
	Permission string `yaml:"permission"`
}
type GHRepoSettingCheck struct {
	Field string `yaml:"field"`
	Value string `yaml:"value"`
}
type CompositeCheck struct {
	Operator string      `yaml:"operator"`
	Checks   []CheckSpec `yaml:"checks"`
}

type ScopeSpec struct {
	Match   ScopeMatch   `yaml:"match"`
	Exclude ScopeExclude `yaml:"exclude"`
}
type ScopeMatch struct {
	Topics      []string `yaml:"topics"`
	Languages   []string `yaml:"languages"`
	NamePattern string   `yaml:"name_pattern"`
}
type ScopeExclude struct {
	Repos       []string `yaml:"repos"`
	NamePattern string   `yaml:"name_pattern"`
}

type Remediation struct {
	Description string `yaml:"description"`
	Link        string `yaml:"link"`
}

// Rule is the parsed representation of kind:Rule YAML.
type Rule struct {
	ID          string
	Name        string
	Severity    Severity
	Tags        []string
	Type        RuleType
	Scope       ScopeSpec
	Check       CheckSpec
	Remediation Remediation
}

// RuleSet is a named collection of rule IDs.
type RuleSet struct {
	ID    string
	Name  string
	Rules []string // rule IDs
}

// Profile binds RuleSets to repos and allows per-repo overrides.
type Profile struct {
	ID        string
	Name      string
	Match     ScopeMatch
	RuleSets  []string // RuleSet IDs
	Overrides []ProfileOverride
}
type ProfileOverride struct {
	Repo    string   `yaml:"repo"`
	Disable []string `yaml:"disable"`
}
