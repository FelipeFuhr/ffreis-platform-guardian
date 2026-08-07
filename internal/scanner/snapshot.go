package scanner

import "github.com/ffreis/platform-guardian/internal/hcl"

type BranchProtection struct {
	RequirePRReviews    bool
	RequireStatusChecks bool
	// Unknown is true when the API call to check branch protection failed
	// for a reason other than "no protection configured" (404) — e.g. a 403
	// from an insufficiently scoped token. Distinguishes "verified: no
	// protection" from "could not verify" so checkers don't report a false
	// policy failure for an inconclusive result.
	Unknown bool
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
	Ref              string
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
