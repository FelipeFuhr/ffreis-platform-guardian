package check

import (
	"fmt"
	"path"
	"regexp"

	"github.com/ffreis/platform-guardian/internal/scanner"
)

// FileExistsChecker checks that a file matching the path pattern exists in FilePaths.
type FileExistsChecker struct {
	Path string
}

func (c *FileExistsChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, fp := range snap.FilePaths {
		matched, err := path.Match(c.Path, fp)
		if err == nil && matched {
			return Result{
				Status:   Pass,
				Message:  fmt.Sprintf("file %s exists", fp),
				Evidence: []string{fp},
			}
		}
		// Also try matching just the base name for patterns without directory
		if matched2, err2 := path.Match(c.Path, path.Base(fp)); err2 == nil && matched2 {
			return Result{
				Status:   Pass,
				Message:  fmt.Sprintf("file %s exists", fp),
				Evidence: []string{fp},
			}
		}
	}
	return Result{
		Status:  Fail,
		Message: fmt.Sprintf("file matching %q not found", c.Path),
	}
}

// FileAbsentChecker checks that no file matching the path pattern exists.
type FileAbsentChecker struct {
	Path string
}

func (c *FileAbsentChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	for _, fp := range snap.FilePaths {
		matched, err := path.Match(c.Path, fp)
		if err == nil && matched {
			return Result{
				Status:   Fail,
				Message:  fmt.Sprintf("file %s should be absent but exists", fp),
				Evidence: []string{fp},
			}
		}
	}
	return Result{
		Status:  Pass,
		Message: fmt.Sprintf("file matching %q is absent as expected", c.Path),
	}
}

// FileContainsChecker checks that a file's content matches a regex pattern.
type FileContainsChecker struct {
	Path    string
	Pattern string
}

func (c *FileContainsChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	content, ok := snap.FileContents[c.Path]
	if !ok {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("file %s not available in snapshot", c.Path),
		}
	}

	re, err := regexp.Compile(c.Pattern)
	if err != nil {
		return Result{
			Status:  Error,
			Message: fmt.Sprintf("invalid pattern %q: %v", c.Pattern, err),
		}
	}

	if re.MatchString(content) {
		return Result{
			Status:  Pass,
			Message: fmt.Sprintf("file %s contains pattern %q", c.Path, c.Pattern),
		}
	}

	return Result{
		Status:   Fail,
		Message:  fmt.Sprintf("file %s does not contain pattern %q", c.Path, c.Pattern),
		Evidence: []string{fmt.Sprintf("file length: %d bytes", len(content))},
	}
}

// FileNotContainsChecker checks that a file's content does NOT match a regex pattern.
type FileNotContainsChecker struct {
	Path    string
	Pattern string
}

func (c *FileNotContainsChecker) Evaluate(snap *scanner.RepoSnapshot) Result {
	content, ok := snap.FileContents[c.Path]
	if !ok {
		return Result{
			Status:  Skip,
			Message: fmt.Sprintf("file %s not available in snapshot", c.Path),
		}
	}

	re, err := regexp.Compile(c.Pattern)
	if err != nil {
		return Result{
			Status:  Error,
			Message: fmt.Sprintf("invalid pattern %q: %v", c.Pattern, err),
		}
	}

	if re.MatchString(content) {
		return Result{
			Status:  Fail,
			Message: fmt.Sprintf("file %s contains forbidden pattern %q", c.Path, c.Pattern),
		}
	}

	return Result{
		Status:  Pass,
		Message: fmt.Sprintf("file %s does not contain forbidden pattern %q", c.Path, c.Pattern),
	}
}
