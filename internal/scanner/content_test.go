package scanner

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestFetchFile_Base64Decodes(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	want := "hello world"
	encoded := base64.StdEncoding.EncodeToString([]byte(want))
	withNewlines := encoded[:5] + "\\n" + encoded[5:] + "\\n"

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/contents/README.md") {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return httpResponse(http.StatusOK, `{"content":"`+withNewlines+`","encoding":"base64"}`), nil
	})

	got, err := FetchFile(context.Background(), "t", "org/repo", "README.md")
	if err != nil {
		t.Fatalf("FetchFile() error = %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFetchFile_404ReturnsEmpty(t *testing.T) {
	origTransport := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	http.DefaultClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusNotFound, ""), nil
	})

	got, err := FetchFile(context.Background(), "", "org/repo", "missing.txt")
	if err != nil {
		t.Fatalf("FetchFile() error = %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestContentScanner_ScanIsNoOp(t *testing.T) {
	t.Parallel()

	snap := NewSnapshot("org/repo")
	s := NewContentScanner(snap)
	if err := s.Scan(context.Background(), "", "org/repo"); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
}
