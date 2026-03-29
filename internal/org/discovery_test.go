package org

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDiscover_FiltersAndPagination(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		page := queryPage(req.URL.RawQuery)

		var body []byte
		switch page {
		case 1:
			// One matching repo, one archived (filtered out).
			body = []byte(`[
  {"full_name":"acme/repo-1","topics":["terraform"],"language":"Go","archived":false,"fork":false},
  {"full_name":"acme/repo-archived","topics":["terraform"],"language":"Go","archived":true,"fork":false}
]`)
		case 2:
			// Empty page ends pagination.
			body = []byte(`[]`)
		default:
			t.Fatalf("unexpected page=%d", page)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	repos, err := Discover(context.Background(), DiscoveryOptions{
		Org:             "acme",
		IncludeArchived: false,
		IncludeForks:    false,
		Topics:          []string{"terraform"},
		NamePattern:     "repo-*",
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d: %#v", len(repos), repos)
	}
	if repos[0].FullName != "acme/repo-1" {
		t.Errorf("unexpected repo: %q", repos[0].FullName)
	}
}

func TestMatchesFilters_TopicOR(t *testing.T) {
	info := RepoInfo{FullName: "acme/repo", Topics: []string{"a", "b"}}
	opts := DiscoveryOptions{IncludeArchived: true, IncludeForks: true, Topics: []string{"x", "b"}}
	if !matchesFilters(info, opts) {
		t.Error("expected match when any filter topic is present")
	}
}

func TestMatchesNamePattern_InvalidPatternDoesNotMatch(t *testing.T) {
	if matchesNamePattern("acme/repo", "[") {
		t.Error("expected invalid glob to not match")
	}
}

func queryPage(rawQuery string) int {
	for _, part := range strings.Split(rawQuery, "&") {
		if strings.HasPrefix(part, "page=") {
			n, err := strconv.Atoi(strings.TrimPrefix(part, "page="))
			if err == nil {
				return n
			}
		}
	}
	return 0
}
