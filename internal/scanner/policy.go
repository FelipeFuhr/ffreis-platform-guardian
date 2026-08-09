package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiReturnedFmt = "API returned %d"

// PolicyScanner fetches branch protection and repo settings via GitHub API.
type PolicyScanner struct {
	snapshot *RepoSnapshot
	warnings io.Writer
}

func NewPolicyScanner(snap *RepoSnapshot, warnings io.Writer) *PolicyScanner {
	if warnings == nil {
		warnings = io.Discard
	}
	return &PolicyScanner{snapshot: snap, warnings: warnings}
}

func (s *PolicyScanner) Type() ScannerType {
	return ScannerTypePolicy
}

func (s *PolicyScanner) Scan(ctx context.Context, token, repo string) error {
	if err := s.fetchRepoSettings(ctx, token, repo); err != nil {
		return fmt.Errorf("fetching repo settings: %w", err)
	}

	// Fetch branch protection for default branch
	defaultBranch := s.snapshot.Settings.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if err := s.fetchBranchProtection(ctx, token, repo, defaultBranch); err != nil {
		// Not fatal
		// scan-fix(golangci:errcheck): explicit best-effort discard — this is
		// itself a non-fatal diagnostic write to a warnings sink (io.Discard by
		// default); a write failure here must never break the scan.
		_, _ = fmt.Fprintf(s.warnings, "warning: could not fetch branch protection for %s/%s: %v\n", repo, defaultBranch, err)
	}

	// Fetch team permissions
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		org := parts[0]
		if err := s.fetchTeamPermissions(ctx, token, org, repo); err != nil {
			// scan-fix(golangci:errcheck): explicit best-effort discard, see above.
			_, _ = fmt.Fprintf(s.warnings, "warning: could not fetch team permissions: %v\n", err)
		}
	}

	return nil
}

func (s *PolicyScanner) fetchRepoSettings(ctx context.Context, token, repo string) error {
	url := fmt.Sprintf(githubAPIBaseURL+"/repos/%s", repo)

	// scan-fix(sonar:duplication): request building extracted to githubGET, see
	// its doc comment in http.go.
	resp, err := githubGET(ctx, token, url, acceptGitHubV3JSON)
	if err != nil {
		return err
	}
	// scan-fix(golangci:errcheck): explicit best-effort discard — a Close() failure
	// on an already-fully-read response body is not actionable here.
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(apiReturnedFmt, resp.StatusCode)
	}

	var result struct {
		DefaultBranch    string `json:"default_branch"`
		Private          bool   `json:"private"`
		AllowSquashMerge bool   `json:"allow_squash_merge"`
		AllowMergeCommit bool   `json:"allow_merge_commit"`
		AllowRebaseMerge bool   `json:"allow_rebase_merge"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	s.snapshot.Settings = RepoSettings{
		DefaultBranch:    result.DefaultBranch,
		Private:          result.Private,
		AllowSquashMerge: result.AllowSquashMerge,
		AllowMergeCommit: result.AllowMergeCommit,
		AllowRebaseMerge: result.AllowRebaseMerge,
	}
	return nil
}

func (s *PolicyScanner) fetchBranchProtection(ctx context.Context, token, repo, branch string) error {
	url := fmt.Sprintf(githubAPIBaseURL+"/repos/%s/branches/%s/protection", repo, branch)

	// scan-fix(sonar:duplication): request building extracted to githubGET, see
	// its doc comment in http.go.
	resp, err := githubGET(ctx, token, url, acceptGitHubV3JSON)
	if err != nil {
		return err
	}
	// scan-fix(golangci:errcheck): explicit best-effort discard — a Close() failure
	// on an already-fully-read response body is not actionable here.
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// No protection configured — record empty
		s.snapshot.BranchProtection[branch] = BranchProtection{}
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		// Could not verify (e.g. 403 from a narrowly scoped token) — record
		// as unknown rather than leaving the map entry absent, so the
		// checker can tell this apart from a confirmed "no protection".
		s.snapshot.BranchProtection[branch] = BranchProtection{Unknown: true}
		return fmt.Errorf(apiReturnedFmt, resp.StatusCode)
	}

	var result struct {
		RequiredPullRequestReviews *struct {
			RequiredApprovingReviewCount int `json:"required_approving_review_count"`
		} `json:"required_pull_request_reviews"`
		RequiredStatusChecks *struct {
			Strict   bool     `json:"strict"`
			Contexts []string `json:"contexts"`
		} `json:"required_status_checks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	bp := BranchProtection{
		RequirePRReviews:    result.RequiredPullRequestReviews != nil,
		RequireStatusChecks: result.RequiredStatusChecks != nil,
	}
	s.snapshot.BranchProtection[branch] = bp
	return nil
}

func (s *PolicyScanner) fetchTeamPermissions(ctx context.Context, token, org, repo string) error {
	url := fmt.Sprintf(githubAPIBaseURL+"/orgs/%s/teams?per_page=100", org)

	// scan-fix(sonar:duplication): request building extracted to githubGET, see
	// its doc comment in http.go.
	resp, err := githubGET(ctx, token, url, acceptGitHubV3JSON)
	if err != nil {
		return err
	}
	// scan-fix(golangci:errcheck): explicit best-effort discard — a Close() failure
	// on an already-fully-read response body is not actionable here.
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(apiReturnedFmt, resp.StatusCode)
	}

	var teams []struct {
		Slug string `json:"slug"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return err
	}

	// Fetch permission for each team
	for _, team := range teams {
		perm, err := s.fetchTeamRepoPermission(ctx, token, org, team.Slug, repo)
		if err != nil {
			continue
		}
		if perm != "" {
			s.snapshot.TeamPermissions[team.Slug] = TeamPermission{Permission: perm}
		}
	}

	return nil
}

func (s *PolicyScanner) fetchTeamRepoPermission(ctx context.Context, token, org, teamSlug, repo string) (string, error) {
	url := fmt.Sprintf(githubAPIBaseURL+"/orgs/%s/teams/%s/repos/%s", org, teamSlug, repo)

	// scan-fix(sonar:duplication): request building extracted to githubGET, see
	// its doc comment in http.go.
	resp, err := githubGET(ctx, token, url, acceptGitHubV3RepoJSON)
	if err != nil {
		return "", err
	}
	// scan-fix(golangci:errcheck): explicit best-effort discard — a Close() failure
	// on an already-fully-read response body is not actionable here.
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(apiReturnedFmt, resp.StatusCode)
	}

	var result struct {
		Permissions struct {
			Admin    bool `json:"admin"`
			Maintain bool `json:"maintain"`
			Push     bool `json:"push"`
			Triage   bool `json:"triage"`
			Pull     bool `json:"pull"`
		} `json:"permissions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	switch {
	case result.Permissions.Admin:
		return "admin", nil
	case result.Permissions.Maintain:
		return "maintain", nil
	case result.Permissions.Push:
		return "write", nil
	case result.Permissions.Triage:
		return "triage", nil
	case result.Permissions.Pull:
		return "read", nil
	}
	return "", nil
}
