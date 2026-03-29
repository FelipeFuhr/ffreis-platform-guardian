package scanner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ContentScanner fetches individual file contents from the GitHub API.
type ContentScanner struct {
	snapshot *RepoSnapshot
}

func NewContentScanner(snap *RepoSnapshot) *ContentScanner {
	return &ContentScanner{snapshot: snap}
}

func (s *ContentScanner) Type() ScannerType {
	return ScannerTypeContent
}

func (s *ContentScanner) Scan(ctx context.Context, token, repo string) error {
	// Content scanner is lazy — individual files are fetched on demand.
	// This Scan is a no-op; use FetchFile for actual fetching.
	return nil
}

// FetchFile fetches a single file's content from the GitHub API.
// Returns empty string and no error on 404.
func FetchFile(ctx context.Context, token, repo, path string) (string, error) {
	url := fmt.Sprintf(githubAPIBaseURL+"/repos/%s/contents/%s", repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	if token != "" {
		req.Header.Set(httpHeaderAuthorization, authBearerPrefix+token)
	}
	req.Header.Set(httpHeaderAccept, acceptGitHubV3JSON)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, url)
	}

	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if result.Encoding == "base64" {
		// GitHub's base64 may have newlines
		cleaned := strings.ReplaceAll(result.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return "", fmt.Errorf("decoding base64: %w", err)
		}
		return string(decoded), nil
	}

	return result.Content, nil
}
