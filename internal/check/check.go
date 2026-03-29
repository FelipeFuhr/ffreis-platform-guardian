package check

import "github.com/ffreis/platform-guardian/internal/scanner"

type Status string

const (
	Pass  Status = "pass"
	Fail  Status = "fail"
	Skip  Status = "skip"
	Error Status = "error"
)

type Result struct {
	Status      Status
	Message     string
	Evidence    []string
	Remediation string
}

type Checker interface {
	Evaluate(snap *scanner.RepoSnapshot) Result
}
