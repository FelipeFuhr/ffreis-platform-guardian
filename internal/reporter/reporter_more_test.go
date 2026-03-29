package reporter

import (
	"bytes"
	"testing"
)

func TestNewReporter_AllFormats(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	cases := []struct {
		format string
	}{
		{format: ""},
		{format: "table"},
		{format: "summary"},
		{format: "json"},
		{format: "sarif"},
		{format: "github-annotations"},
		{format: "annotations"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()

			rep, err := New(tc.format, &buf)
			if err != nil {
				t.Fatalf("New(%q) error = %v", tc.format, err)
			}
			if rep == nil {
				t.Fatalf("expected reporter")
			}
		})
	}
}
