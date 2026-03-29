package scanner

import "github.com/ffreis/platform-guardian/internal/hcl"

type BranchProtection struct {
	RequirePRReviews    bool
	RequireStatusChecks bool
}

type TeamPermission struct {
	Permission string // read, write, admin, maintain, triage
}

type RepoSettings struct {
	AllowSquashMerge bool
	AllowMergeCommit bool
	AllowRebaseMerge bool
	DefaultBranch    string
	Private          bool
}

type RepoSnapshot struct {
	Repo             string
	Topics           []string
	Languages        []string
	FilePaths        []string
	FileContents     map[string]string
	TFModules        []hcl.TFModule
	BranchProtection map[string]BranchProtection
	TeamPermissions  map[string]TeamPermission
	Settings         RepoSettings
}

func NewSnapshot(repo string) *RepoSnapshot {
	return &RepoSnapshot{
		Repo:             repo,
		FileContents:     make(map[string]string),
		BranchProtection: make(map[string]BranchProtection),
		TeamPermissions:  make(map[string]TeamPermission),
	}
}
