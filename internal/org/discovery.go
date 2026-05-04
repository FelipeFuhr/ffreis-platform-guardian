package org

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

type DiscoveryOptions struct {
	Token           string
	Org             string
	IncludeArchived bool
	IncludeForks    bool
	Topics          []string
	NamePattern     string // glob
}

type RepoInfo struct {
	FullName string // "org/repo"
	Topics   []string
	Language string
	Archived bool
	Fork     bool
}

type rawRepo struct {
	FullName string   `json:"full_name"`
	Topics   []string `json:"topics"`
	Language string   `json:"language"`
	Archived bool     `json:"archived"`
	Fork     bool     `json:"fork"`
}

// Discover pages through GET /orgs/{org}/repos and returns matching repos.
func Discover(ctx context.Context, opts DiscoveryOptions) ([]RepoInfo, error) {
	var repos []RepoInfo
	page := 1

	for {
		rawRepos, isLastPage, err := fetchOrgReposPage(ctx, opts.Org, opts.Token, page)
		if err != nil {
			return nil, err
		}
		if len(rawRepos) == 0 {
			break
		}

		for _, r := range rawRepos {
			info := RepoInfo(r)

			if !matchesFilters(info, opts) {
				continue
			}

			repos = append(repos, info)
		}

		if isLastPage {
			break
		}
		page++
	}

	return repos, nil
}

func fetchOrgReposPage(ctx context.Context, org, token string, page int) ([]rawRepo, bool, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all&page=%d", org, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("creating request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("fetching repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, false, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var rawRepos []rawRepo
	if err := json.NewDecoder(resp.Body).Decode(&rawRepos); err != nil {
		return nil, false, fmt.Errorf("decoding response: %w", err)
	}

	return rawRepos, isLastPage(rawRepos), nil
}

func isLastPage(rawRepos []rawRepo) bool {
	// GitHub returns fewer than per_page when at last page.
	return len(rawRepos) < 100
}

func matchesFilters(info RepoInfo, opts DiscoveryOptions) bool {
	if !opts.IncludeArchived && info.Archived {
		return false
	}
	if !opts.IncludeForks && info.Fork {
		return false
	}
	if len(opts.Topics) > 0 && !hasAnyTopic(info.Topics, opts.Topics) {
		return false
	}
	if opts.NamePattern != "" && !matchesNamePattern(info.FullName, opts.NamePattern) {
		return false
	}
	return true
}

func hasAnyTopic(repoTopics, filterTopics []string) bool {
	if len(filterTopics) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(repoTopics))
	for _, t := range repoTopics {
		set[t] = struct{}{}
	}
	for _, t := range filterTopics {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}

func matchesNamePattern(fullName, pattern string) bool {
	matched, err := path.Match(pattern, repoName(fullName))
	return err == nil && matched
}

func repoName(fullName string) string {
	if idx := strings.LastIndex(fullName, "/"); idx >= 0 {
		return fullName[idx+1:]
	}
	return fullName
}
