package reporter

import (
	"bytes"
	"testing"
)

func TestNewReporter_UnknownFormatErrors(t *testing.T) {
	t.Parallel()

	_, err := New("nope", &bytes.Buffer{})
	if err == nil {
		t.Fatalf("expected error for unknown format")
	}
}
