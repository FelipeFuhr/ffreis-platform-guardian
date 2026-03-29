package scanner

import "context"

type ScannerType string

const (
	ScannerTypeStructure ScannerType = "structure"
	ScannerTypeContent   ScannerType = "content"
	ScannerTypeTerraform ScannerType = "terraform"
	ScannerTypePolicy    ScannerType = "policy"
)

// Scanner collects data about a repository.
type Scanner interface {
	Type() ScannerType
	Scan(ctx context.Context, token, repo string) error
}
