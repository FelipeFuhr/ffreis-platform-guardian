package scanner

import (
	"context"
	"net/http"
	"testing"
)

func TestStructureScanner_AddsBlobPaths(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	snap := NewSnapshot("org/repo")
	s := NewStructureScanner(snap)

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusOK, `{
  "tree": [
    {"path": "README.md", "type": "blob"},
    {"path": "internal", "type": "tree"},
    {"path": "internal/main.go", "type": "blob"}
  ],
  "truncated": false
}`), nil
	})

	if err := s.Scan(context.Background(), "", "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snap.FilePaths) != 2 {
		t.Fatalf("expected 2 blob paths, got %d: %#v", len(snap.FilePaths), snap.FilePaths)
	}
}
