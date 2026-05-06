package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// StructureScanner fetches the repo tree from the GitHub API.
type StructureScanner struct {
	snapshot *RepoSnapshot
	warnings io.Writer
}

func NewStructureScanner(snap *RepoSnapshot, warnings io.Writer) *StructureScanner {
	if warnings == nil {
		warnings = io.Discard
	}
	return &StructureScanner{snapshot: snap, warnings: warnings}
}

func (s *StructureScanner) Type() ScannerType {
	return ScannerTypeStructure
}

func (s *StructureScanner) Scan(ctx context.Context, token, repo string) error {
	ref := s.snapshot.Ref
	if ref == "" {
		ref = "HEAD"
	}
	url := fmt.Sprintf(githubAPIBaseURL+"/repos/%s/git/trees/%s?recursive=1", repo, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if token != "" {
		req.Header.Set(httpHeaderAuthorization, authBearerPrefix+token)
	}
	req.Header.Set(httpHeaderAccept, acceptGitHubV3JSON)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, url)
	}

	var result struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if result.Truncated {
		// Log warning but use what's available
		fmt.Fprintf(s.warnings, "warning: tree response truncated for %s\n", repo)
	}

	for _, item := range result.Tree {
		if item.Type == "blob" {
			s.snapshot.FilePaths = append(s.snapshot.FilePaths, item.Path)
		}
	}

	return nil
}
